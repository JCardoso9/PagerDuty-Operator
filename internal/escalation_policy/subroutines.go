package escalation_policy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_errors"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const escalationPolicyFinalizer = "pagerduty.platform.share-now.com/escalation_policy"
const escalationPolicyReady = "PDEscalationPolicyReady"
const RequeWaitTime = time.Second * 10

type SubroutineHandler struct {
	EscalationPolicy *v1alpha1.EscalationPolicy
	Logger           logr.Logger
	K8sClient        client.Client
	EPAdapter        *EPAdapter
	conditionManager condition.Conditions
}

func (e *SubroutineHandler) ReconcileCreation() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Creation...")

	// no upstream policy has been created yet
	if !e.policyIDExists() {
		e.Logger.Info("Upstream Escalation Policy policy not found. Creating...")

		// Handles creation of upstream policy
		policyID, err := e.EPAdapter.CreateEscalationPolicy(&e.EscalationPolicy.Spec)
		if err != nil {
			e.Logger.Error(err, "Failed to create PD Escalation Policy")
			return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, err, err.Error())
		}

		e.Logger.Info("Updating Escalation Policy status...")
		// TODO: Update the rest of the policy status/spec
		e.EscalationPolicy.Status.PolicyID = policyID
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, nil, "Escalation policy created")
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileDeletion() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Deletion...")

	if e.deletionTimestampExists() {
		e.Logger.Info("Deletion timestamp found. Deleting...")

		if e.policyIDExists() {
			e.Logger.Info("Upstream policy found, making API deletion call for Escalation policy ...")
			if err := e.EPAdapter.DeletePDEscalationPolicy(e.EscalationPolicy.Status.PolicyID); err != nil {
				e.Logger.Error(err, "Failed to delete escalation policy")
				return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, err, err.Error())
			}
		}

		err := e.removeFinalizer()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("No deletion timestamp found. Skipping deletion...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileUpdate() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Escalation Policy Update...")

	if !e.policyIDExists() {
		e.Logger.Info("No upstream policy created yet. Skipping Update...")
		return pd_utils.ContinueProcessing()
	}

	equal, err := e.EPAdapter.EqualToUpstream(*e.EscalationPolicy)
	if err != nil {
		e.Logger.Error(err, "Failed to compare Escalation Policy spec with upstream policy")
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, err, err.Error())
	}

	if !equal {
		e.Logger.Info("Escalation Policy spec does not match upstream service. Updating...")
		err := e.EPAdapter.UpdatePDEscalationPolicy(e.EscalationPolicy)

		if err != nil {
			e.Logger.Error(err, "Failed to update Escalation Policy")
			return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, err, err.Error())
		}

		e.Logger.Info("Escalation Policy changed...")
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, escalationPolicyReady, nil, "PagerDuty Escalation Policy matches upstream service")
	}

	e.Logger.Info("PagerDuty Escalation Policy not changed, Reconcile Update PagerDuty Escalation Policy done...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) policyIDExists() bool {
	return e.EscalationPolicy.Status.PolicyID != ""
}

func (e *SubroutineHandler) deletionTimestampExists() bool {
	return !e.EscalationPolicy.GetDeletionTimestamp().IsZero()
}

func (e *SubroutineHandler) AddFinalizer() (pd_utils.OperationResult, error) {
	if !controllerutil.ContainsFinalizer(e.EscalationPolicy, escalationPolicyFinalizer) {
		e.Logger.Info("Adding Finalizer for Escalation policy")
		if ok := controllerutil.AddFinalizer(e.EscalationPolicy, escalationPolicyFinalizer); !ok {
			e.Logger.Info("Could not add finalizer for Escalation policy")
			return pd_utils.Requeue()
		}

		err := e.K8sClient.Update(context.Background(), e.EscalationPolicy)
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) removeFinalizer() error {
	if !controllerutil.ContainsFinalizer(e.EscalationPolicy, escalationPolicyFinalizer) {
		e.Logger.Info("No Finalizer present, skipping finalizer removal...")
		return nil
	}

	e.Logger.Info("Removing Finalizer for Escalation Policy after successfully perform the operations")
	if ok := controllerutil.RemoveFinalizer(e.EscalationPolicy, escalationPolicyFinalizer); !ok {
		e.Logger.Info("Failed to remove finalizer for Escalation Policy")
		return errors.New("failed to remove finalizer for Escalation Policy")
	}

	return e.K8sClient.Update(context.TODO(), e.EscalationPolicy)
}

// StatusUpdate updates the project claim status
func (e *SubroutineHandler) StatusUpdate() error {
	e.Logger.Info("Updating status...")
	if err := e.K8sClient.Status().Update(context.TODO(), e.EscalationPolicy); err != nil {
		e.Logger.Error(err, "Failed to update EscalationPolicy status")
		return pd_errors.Wrap(err, fmt.Sprintf("failed to update EscalationPolicy state for %s", e.EscalationPolicy.Name))
	}

	e.Logger.Info("Status updated...")
	return nil
}

func (e *SubroutineHandler) SetEscalationPolicyCondition(conditionType v1alpha1.ConditionType, reason string, err error, message string) (pd_utils.OperationResult, error) {
	conditions := &e.EscalationPolicy.Status.Conditions

	// defer e.StatusUpdate()??

	if err != nil {
		e.Logger.Info("Setting ready condition to false", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message, "error", err.Error())
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, err.Error())

		err := e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update EscalationPolicy to false ready condition, Requeing in 10 seconds... ")
		}

		return pd_utils.RequeueAfter(RequeWaitTime, err)
	}

	// TODO: Can probably simplify this if by only setting condition if on of these conditions is false...
	if len(*conditions) > 0 && (*conditions)[0].Status == metav1.ConditionTrue && (*conditions)[0].Message == message {
		err = e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update EscalationPolicy to true ready condition ")
			return pd_utils.RequeueAfter(RequeWaitTime, err)
		}
		return pd_utils.StopProcessing()
	}

	e.Logger.Info("Setting ready condition to true", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message)
	e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionTrue, reason, message)
	err = e.StatusUpdate()

	return pd_utils.RequeueOnErrorOrStop(err)
}

func (e *SubroutineHandler) Initialization() (pd_utils.OperationResult, error) {
	e.Logger.Info("Starting Initialization...")
	if e.EscalationPolicy.Status.Conditions == nil {
		e.EscalationPolicy.Status.Conditions = []metav1.Condition{}

		err := e.StatusUpdate()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("Initialization done...")
	return pd_utils.ContinueProcessing()
}
