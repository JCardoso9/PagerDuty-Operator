/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package business_service

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pagerdutyalpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/k8s_utils"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
)

// BusinessServiceReconciler reconciles a BusinessService object
type BusinessServiceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type Subroutines interface {
	ReconcileCreation() (pd_utils.OperationResult, error)
	ReconcileDeletion() (pd_utils.OperationResult, error)
	ReconcileUpdate() (pd_utils.OperationResult, error)
	Initialization() (pd_utils.OperationResult, error)
	AddFinalizer() (pd_utils.OperationResult, error)
}

//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=businessservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=businessservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=businessservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BusinessService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BusinessServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting Business Service reconcile...")

	businessService := &pagerdutyalpha1.BusinessService{}
	err := r.Get(ctx, req.NamespacedName, businessService)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("Business Service resource not found. Ignoring since object must be deleted")
			return k8s_utils.DoNotRequeue()
		}
		// Error reading the object - requeue the request.
		log.Info("Failed to get Business Service")

		return k8s_utils.RequeueWithError(err)
	}

	subroutineHandler := &SubroutineHandler{
		Logger:    log.WithName("business-service controller"),
		K8sClient: r.Client,
		BSAdapter: &BSAdapter{
			Logger:    log,
			PD_Client: pagerduty.NewClient(""),
		},
		BusinessService:  businessService,
		conditionManager: condition.NewConditionManager(),
	}

	result, err := r.ReconcileHandler(subroutineHandler)
	// reason := "ReconcileError"
	// _, _ = adapter.SetProjectClaimCondition(gcpv1alpha1.ConditionError, reason, err)

	return result, err
}

type ReconcileOperation func() (pd_utils.OperationResult, error)

func (r *BusinessServiceReconciler) ReconcileHandler(subroutines Subroutines) (ctrl.Result, error) {
	operations := []ReconcileOperation{
		subroutines.Initialization,
		subroutines.AddFinalizer,
		subroutines.ReconcileDeletion,
		subroutines.ReconcileCreation,
		subroutines.ReconcileUpdate,
	}
	for _, operation := range operations {
		result, err := operation()
		if err != nil || result.RequeueRequest {
			return ctrl.Result{RequeueAfter: result.RequeueDelay}, err
		}
		if result.CancelRequest {
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BusinessServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pagerdutyalpha1.BusinessService{}).
		Complete(r)
}
