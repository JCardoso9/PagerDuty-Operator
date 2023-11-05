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

package pdservice

import (
	"context"
	"reflect"

	"github.com/PagerDuty/go-pagerduty"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pagerdutyalpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	k8s_utils "gitlab.share-now.com/platform/pagerduty-operator/internal/k8s_utils"
)

const service_finalizer = "pagerduty.platform.share-now.com/pagerdutyservice"

// PagerdutyServiceReconciler reconciles a PagerdutyService object
type PagerdutyServiceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions (RBAC) on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=pagerdutyservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=pagerdutyservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=pagerdutyservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PagerdutyServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconcile...")

	pdService := &pagerdutyalpha1.PagerdutyService{}
	err := r.Get(ctx, req.NamespacedName, pdService)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("pagerdutyservice resource not found. Ignoring since object must be deleted")
			return k8s_utils.DoNotRequeue()
		}
		// Error reading the object - requeue the request.
		log.Info("Failed to get pagerdutyservice")

		return k8s_utils.RequeueWithError(err)
	}

	// If the deployment is being deleted
	log.Info("Checking for deletion timestamp...")
	if !pdService.GetDeletionTimestamp().IsZero() {
		log.Info("PD Service was marked for deletion.")

		if controllerutil.ContainsFinalizer(pdService, service_finalizer) {

			// Delete associated kubernetes secret
			log.Info("Deleting PDService ...")
			if err = r.deletePDService(pdService, ctx, req); err != nil {
				return k8s_utils.RequeueWithError(err)
			}
			log.Info("PDService deleted ...")

			log.Info("Removing Finalizer for PD Service after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(pdService, service_finalizer); !ok {
				log.Error(err, "Failed to remove finalizer for PD Service")
				return k8s_utils.Requeue()
			}

			if err := r.Update(ctx, pdService); err != nil {
				log.Error(err, "Failed to remove finalizer for Escalation Policy")
				return k8s_utils.RequeueWithError(err)
			}
		}

		return k8s_utils.DoNotRequeue()
	}

	if !controllerutil.ContainsFinalizer(pdService, service_finalizer) {
		log.Info("Adding Finalizer for PD Service  policy")
		if ok := controllerutil.AddFinalizer(pdService, service_finalizer); !ok {
			log.Error(err, "Failed to add finalizer into the PD Service custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, pdService); err != nil {
			log.Error(err, "Failed to update PD Service custom resource to add finalizer")
			return k8s_utils.RequeueWithError(err)
		}

		return k8s_utils.RequeueWithError(err)
	}

	// no upstream PD service has been created yet
	if pdService.Status.ServiceID == "" {
		log.Info("No upstream PD service created yet. Creating...")

		// Handles creation of pagerduty service
		serviceID, err := r.handlePDService(pdService, ctx, req)
		if err != nil {
			log.Error(err, "Failed to handle PD Service")
			return k8s_utils.RequeueWithError(err)
		}

		log.Info("Checking if status update necessary...", "ServiceID", serviceID)
		// Update PD Service Id
		if !reflect.DeepEqual(serviceID, pdService.Status.ServiceID) {
			pdService.Status.ServiceID = serviceID
			log.Info("Updating status...", "ServiceID", serviceID)
			err := r.Status().Update(ctx, pdService)
			log.Info("Status updated...", "policy.Status.ServiceID", pdService.Status.ServiceID)
			if err != nil {
				log.Error(err, "Failed to update PD Service status")
				return k8s_utils.RequeueWithError(err)
			}

			return k8s_utils.DoNotRequeue()
		}
	}

	return k8s_utils.DoNotRequeue()
}

func (r *PagerdutyServiceReconciler) handlePDService(pdService *pagerdutyalpha1.PagerdutyService, ctx context.Context, req ctrl.Request) (string, error) {
	log := log.FromContext(ctx)

	if pdService.Status.ServiceID != "" {
		log.Info("PD Service already exists. Skipping PD Service creation...")
		return pdService.Status.ServiceID, nil
	}

	client := pagerduty.NewClient("")
	res, err := client.CreateServiceWithContext(ctx, pdService.Spec.Convert())
	if err != nil {
		log.Error(err, "PD Service creation unsuccessfull...")
	}

	return res.ID, nil
}

func (r *PagerdutyServiceReconciler) deletePDService(pdService *pagerdutyalpha1.PagerdutyService, ctx context.Context, req ctrl.Request) error {
	log := log.FromContext(ctx)
	log.Info("Deleting PD Service...")

	if pdService.Status.ServiceID == "" {
		log.Info("ERROR: No PD Service found. Skipping PD Service deletion...")
		return nil
	}

	client := pagerduty.NewClient("")
	err := client.DeleteServiceWithContext(ctx, pdService.Status.ServiceID)
	if err != nil {
		log.Error(err, "ERROR: Failed to delete PD Service")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
// Note that the Deployment will be also watched in order to ensure its
// desirable state on the cluster
func (r *PagerdutyServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pagerdutyalpha1.PagerdutyService{}).
		Complete(r)
}
