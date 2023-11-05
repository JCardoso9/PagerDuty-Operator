package escalation_policy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	pdv1alpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_errors"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const escalationPolicyFinalizer = "pagerduty.platform.share-now.com/escalationpolicy"
const escalationPolicyReady = "EscalationPolicyReady"

type EscalationPolicyAdapter struct {
	EscalationPolicy *v1alpha1.EscalationPolicy
	Logger           logr.Logger
	K8sClient        client.Client
	PD_Client        *pagerduty.Client
	conditionManager condition.Conditions
}

func (e *EscalationPolicyAdapter) ReconcileCreation() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Creation...")

	// no upstream policy has been created yet
	if !e.policyIDExists() {
		e.Logger.Info("Upstream policy not found. Creating...")

		// Handles creation of upstream policy
		policyID, err := e.createEscalationPolicy()
		if err != nil {
			e.Logger.Error(err, "Failed to create PD Escalation Policy")
			return e.SetEscalationPolicyCondition(pdv1alpha1.ConditionReady, escalationPolicyReady, err, "")
		}

		e.Logger.Info("Updating status...")
		// TODO: Update the rest of the policy status/spec
		e.EscalationPolicy.Status.PolicyID = policyID
		return e.SetEscalationPolicyCondition(pdv1alpha1.ConditionReady, escalationPolicyReady, nil, "Escalation policy created")
	}

	return pd_utils.ContinueProcessing()
}

func (e *EscalationPolicyAdapter) createEscalationPolicy() (string, error) {

	if e.EscalationPolicy.Status.PolicyID != "" {
		e.Logger.Info("Policy already exists. Skipping Escalation policy creation...")
		return e.EscalationPolicy.Status.PolicyID, nil
	}

	res, err := e.PD_Client.CreateEscalationPolicyWithContext(context.TODO(), e.EscalationPolicy.Spec.Convert())
	if err != nil {
		e.Logger.Error(err, "Escalation policy creation unsuccessfull...")
		return "", err
	}

	return res.ID, nil
}

func (e *EscalationPolicyAdapter) ReconcileDeletion() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Deletion...")

	if e.deletionTimestampExists() {
		e.Logger.Info("Deletion timestamp found. Deleting...")

		if e.policyIDExists() {
			e.Logger.Info("Upstream policy found, making API deletion call for Escalation policy ...")
			if err := e.deletePDEscalationPolicy(); err != nil {
				e.Logger.Error(err, "Failed to delete escalation policy")
				return e.SetEscalationPolicyCondition(pdv1alpha1.ConditionReady, escalationPolicyReady, err, "")
			}
		}

		err := e.removeFinalizer()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("No deletion timestamp found. Skipping deletion...")
	return pd_utils.ContinueProcessing()
}

func (e *EscalationPolicyAdapter) deletePDEscalationPolicy() error {
	e.Logger.Info("Deleting policy...")

	err := e.PD_Client.DeleteEscalationPolicyWithContext(context.TODO(), e.EscalationPolicy.Status.PolicyID)
	if err != nil {
		e.Logger.Error(err, "ERROR: Failed to delete Escalation policy")
		return err
	}

	e.Logger.Info("Policy deleted...")
	return nil
}

func (e *EscalationPolicyAdapter) ReconcileUpdate() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Update...")

	if !e.policyIDExists() {
		e.Logger.Info("No upstream policy created yet. Skipping Update...")
		return pd_utils.ContinueProcessing()
	}

	changed, err := e.updatePDEscalationPolicy()
	if err != nil {
		e.Logger.Error(err, "Failed to update Escalation policy")
		return e.SetEscalationPolicyCondition(pdv1alpha1.ConditionReady, escalationPolicyReady, err, "")
	}

	e.Logger.Info("Reconcile Update done...")
	if changed {
		e.Logger.Info("Changed...")
		return e.SetEscalationPolicyCondition(pdv1alpha1.ConditionReady, escalationPolicyReady, nil, "Escalation policy matches upstream policy")
	}

	e.Logger.Info("Not changed...")
	return pd_utils.ContinueProcessing()
}

func (e *EscalationPolicyAdapter) updatePDEscalationPolicy() (bool, error) {
	PDPolicy, err := e.getPDEscalationPolicy(e.EscalationPolicy.Status.PolicyID)
	if err != nil {
		e.Logger.Error(err, "Failed to get Escalation policy")
		return false, err
	}

	k8sPolicy := e.EscalationPolicy.Spec.Convert()

	e.Logger.Info("Comparing Escalation policy spec with upstream policy...")
	if !e.policyEqualUpstream(PDPolicy, k8sPolicy) {
		e.Logger.Info("Updating policy...")
		_, err := e.PD_Client.UpdateEscalationPolicyWithContext(
			context.TODO(),
			e.EscalationPolicy.Status.PolicyID,
			k8sPolicy,
		)

		if err != nil {
			e.Logger.Error(err, "API Failed to update Escalation policy")
			return false, err
		}

		e.Logger.Info("Policy updated...")
		return true, nil
		// Update condition, call status update
	}

	e.Logger.Info("CRD Policy matches upstream...")
	return false, nil
}

func (e *EscalationPolicyAdapter) policyEqualUpstream(upstreamPolicy *pagerduty.EscalationPolicy, k8sPolicy pagerduty.EscalationPolicy) bool {

	return k8sPolicy.Name == upstreamPolicy.Name &&
		k8sPolicy.Description == upstreamPolicy.Description &&
		k8sPolicy.NumLoops == upstreamPolicy.NumLoops &&
		k8sPolicy.OnCallHandoffNotifications == upstreamPolicy.OnCallHandoffNotifications &&
		e.escalationRulesEqual(k8sPolicy, upstreamPolicy.EscalationRules)
}

func (e *EscalationPolicyAdapter) escalationRulesEqual(localPolicy pagerduty.EscalationPolicy, upstreamRules []pagerduty.EscalationRule) bool {

	if len(localPolicy.EscalationRules) != len(upstreamRules) {
		return false
	}

	for i, rule := range localPolicy.EscalationRules {

		if !(rule.Delay == upstreamRules[i].Delay) {
			return false
		}

		if len(rule.Targets) != len(upstreamRules[i].Targets) {
			return false
		}

		for j, target := range rule.Targets {
			if target.ID != upstreamRules[i].Targets[j].ID {
				return false
			}
		}
	}

	return true
}

func (e *EscalationPolicyAdapter) getPDEscalationPolicy(id string) (*pagerduty.EscalationPolicy, error) {
	PDPolicy, err := e.PD_Client.GetEscalationPolicyWithContext(context.TODO(), id, &pagerduty.GetEscalationPolicyOptions{})
	if err != nil {
		e.Logger.Error(err, "Failed to get Escalation policy")
		return nil, err
	}

	e.Logger.Info("Escalation policy retrieved", "PDPolicy", PDPolicy)
	return PDPolicy, nil
}

func (e *EscalationPolicyAdapter) policyIDExists() bool {
	return e.EscalationPolicy.Status.PolicyID != ""
}

func (e *EscalationPolicyAdapter) deletionTimestampExists() bool {
	return !e.EscalationPolicy.GetDeletionTimestamp().IsZero()
}

func (e *EscalationPolicyAdapter) AddFinalizer() (pd_utils.OperationResult, error) {
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

func (e *EscalationPolicyAdapter) removeFinalizer() error {
	if !controllerutil.ContainsFinalizer(e.EscalationPolicy, escalationPolicyFinalizer) {
		e.Logger.Info("No Finalizer present, skipping finalizer removal...")
		return nil
	}

	e.Logger.Info("Removing Finalizer for Escalation Policy after successfully perform the operations")
	if ok := controllerutil.RemoveFinalizer(e.EscalationPolicy, escalationPolicyFinalizer); !ok {
		e.Logger.Info("Failed to remove finalizer for Escalation Policy")
		return errors.New("Failed to remove finalizer for Escalation Policy")
	}

	return e.K8sClient.Update(context.TODO(), e.EscalationPolicy)
}

// StatusUpdate updates the project claim status
func (e *EscalationPolicyAdapter) StatusUpdate() error {
	e.Logger.Info("Updating status...")
	if err := e.K8sClient.Status().Update(context.TODO(), e.EscalationPolicy); err != nil {
		e.Logger.Error(err, "Failed to update EscalationPolicy status")
		return pd_errors.Wrap(err, fmt.Sprintf("failed to update EscalationPolicy state for %s", e.EscalationPolicy.Name))
	}

	e.Logger.Info("Status updated...")
	return nil
}

func (e *EscalationPolicyAdapter) SetEscalationPolicyCondition(conditionType pdv1alpha1.ConditionType, reason string, err error, message string) (pd_utils.OperationResult, error) {
	conditions := &e.EscalationPolicy.Status.Conditions

	if err != nil {
		e.Logger.Info("Setting ready condition to false", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message, "error", err.Error())
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, err.Error())

		err := e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update EscalationPolicy to false ready condition ")
		}

		return pd_utils.RequeueAfter(time.Second*10, err)
	}

	if len(*conditions) > 0 && (*conditions)[0].Status == metav1.ConditionTrue && (*conditions)[0].Message == message {
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, message)
		return pd_utils.StopProcessing()
	}

	e.Logger.Info("Setting ready condition to true", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message)
	e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionTrue, reason, message)
	err = e.StatusUpdate()

	return pd_utils.RequeueOnErrorOrStop(err)
}

func (e *EscalationPolicyAdapter) Initialization() (pd_utils.OperationResult, error) {
	e.Logger.Info("Starting Initialization...")
	if e.EscalationPolicy.Status.Conditions == nil {
		e.EscalationPolicy.Status.Conditions = []metav1.Condition{}

		err := e.StatusUpdate()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("Initialization done...")
	return pd_utils.ContinueProcessing()
}
