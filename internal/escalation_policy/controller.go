package escalation_policy

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
type EscalationPolicyReconciler struct {
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

//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=escalationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=escalationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pagerduty.platform.share-now.com,resources=escalationpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EscalationPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *EscalationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconcile...")

	policy := &pagerdutyalpha1.EscalationPolicy{}
	err := r.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("Escalation policy resource not found. Ignoring since object must be deleted")
			return k8s_utils.DoNotRequeue()
		}
		// Error reading the object - requeue the request.
		log.Info("Failed to get escalation policy")

		return k8s_utils.RequeueWithError(err)
	}

	adapter := &EscalationPolicyAdapter{
		Logger:           log,
		K8sClient:        r.Client,
		PD_Client:        pagerduty.NewClient(""),
		EscalationPolicy: policy,
		conditionManager: condition.NewConditionManager(),
	}

	result, err := r.ReconcileHandler(adapter)
	// reason := "ReconcileError"
	// _, _ = adapter.SetProjectClaimCondition(gcpv1alpha1.ConditionError, reason, err)

	return result, err
}

type ReconcileOperation func() (pd_utils.OperationResult, error)

func (r *EscalationPolicyReconciler) ReconcileHandler(adapter Adapter) (ctrl.Result, error) {
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
func (r *EscalationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pagerdutyalpha1.EscalationPolicy{}).
		Complete(r)
}
