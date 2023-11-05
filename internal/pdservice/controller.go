package pdservice

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/PagerDuty/go-pagerduty"
	pagerdutyalpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	k8s_utils "gitlab.share-now.com/platform/pagerduty-operator/internal/k8s_utils"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
)

// EscalationPolicyReconciler reconciles a EscalationPolicy object
type PagerdutyServiceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type Adapter interface {
	ReconcileCreation() (pd_utils.OperationResult, error)
	ReconcileDeletion() (pd_utils.OperationResult, error)
	ReconcileUpdate() (pd_utils.OperationResult, error)
	Initialization() (pd_utils.OperationResult, error)
	AddFinalizer() (pd_utils.OperationResult, error)
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

	adapter := &PDServiceAdapter{
		Logger:           log,
		K8sClient:        r.Client,
		PD_Client:        pagerduty.NewClient(""),
		PagerdutyService: pdService,
		conditionManager: condition.NewConditionManager(),
	}

	result, err := r.ReconcileHandler(adapter)
	// reason := "ReconcileError"
	// _, _ = adapter.SetProjectClaimCondition(gcpv1alpha1.ConditionError, reason, err)

	return result, err
}

type ReconcileOperation func() (pd_utils.OperationResult, error)

func (r *PagerdutyServiceReconciler) ReconcileHandler(adapter Adapter) (ctrl.Result, error) {
	operations := []ReconcileOperation{
		adapter.Initialization,
		adapter.AddFinalizer,
		adapter.ReconcileDeletion,
		adapter.ReconcileCreation,
		adapter.ReconcileUpdate,
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
func (r *PagerdutyServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pagerdutyalpha1.EscalationPolicy{}).
		Complete(r)
}
