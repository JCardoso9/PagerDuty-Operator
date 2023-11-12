package escalation_policy

import (
	"context"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

type Adapter interface {
	convert(*v1alpha1.EscalationPolicySpec) pagerduty.EscalationPolicy
	CreateEscalationPolicy(k8sPDEscalationPolicy *v1alpha1.EscalationPolicySpec) (string, error)
	GetPDEscalationPolicy(string) (*pagerduty.EscalationPolicy, error)
	DeletePDEscalationPolicy(string) error
	UpdatePDEscalationPolicy(*v1alpha1.EscalationPolicy) error
	EqualToUpstream(v1alpha1.EscalationPolicy) (bool, error)
}

type EPAdapter struct {
	Logger    logr.Logger
	PD_Client *pagerduty.Client
}

var escalation_policy_reference_type = "escalation_policy_reference"

func (adapter *EPAdapter) convert(policy *v1alpha1.EscalationPolicy) pagerduty.EscalationPolicy {
	if policy.Spec.Team == "" {
		fmt.Println("------------------------------------ Team is empty")
		return pagerduty.EscalationPolicy{
			APIObject: pagerduty.APIObject{
				ID:   policy.Status.PolicyID,
				Type: escalation_policy_reference_type,
			},
			Name:                       policy.Spec.Name,
			Description:                policy.Spec.Description,
			OnCallHandoffNotifications: policy.Spec.OnCallHandoffNotifications,
			NumLoops:                   policy.Spec.NumLoops,
			EscalationRules:            policy.Spec.EscalationRules.ConvertToPagerDutyObj(),
		}
	}

	return pagerduty.EscalationPolicy{
		APIObject: pagerduty.APIObject{
			ID:   policy.Status.PolicyID,
			Type: escalation_policy_reference_type,
		},
		Name:                       policy.Spec.Name,
		Description:                policy.Spec.Description,
		OnCallHandoffNotifications: policy.Spec.OnCallHandoffNotifications,
		NumLoops:                   policy.Spec.NumLoops,
		Teams:                      []pagerduty.APIReference{policy.Spec.Team.ToReference()},
		EscalationRules:            policy.Spec.EscalationRules.ConvertToPagerDutyObj(),
	}
}

func (spec *EPAdapter) convertSpec(policy *v1alpha1.EscalationPolicySpec) pagerduty.EscalationPolicy {
	if policy.Team == "" {
		fmt.Println("------------------------------------ Team is empty")
		return pagerduty.EscalationPolicy{
			Name:                       policy.Name,
			Description:                policy.Description,
			OnCallHandoffNotifications: policy.OnCallHandoffNotifications,
			NumLoops:                   policy.NumLoops,
			EscalationRules:            policy.EscalationRules.ConvertToPagerDutyObj(),
		}
	}

	return pagerduty.EscalationPolicy{
		Name:                       policy.Name,
		Description:                policy.Description,
		OnCallHandoffNotifications: policy.OnCallHandoffNotifications,
		NumLoops:                   policy.NumLoops,
		Teams:                      []pagerduty.APIReference{policy.Team.ToReference()},
		EscalationRules:            policy.EscalationRules.ConvertToPagerDutyObj(),
	}
}

func (adapter *EPAdapter) CreateEscalationPolicy(k8sPDEscalationPolicy *v1alpha1.EscalationPolicySpec) (string, error) {

	res, err := adapter.PD_Client.CreateEscalationPolicyWithContext(context.TODO(), adapter.convertSpec(k8sPDEscalationPolicy))
	if err != nil {
		adapter.Logger.Error(err, "Escalation policy creation unsuccessfull...")
		return "", err
	}

	return res.ID, nil
}

func (adapter *EPAdapter) DeletePDEscalationPolicy(id string) error {
	adapter.Logger.Info("Deleting policy...")

	err := adapter.PD_Client.DeleteEscalationPolicyWithContext(context.TODO(), id)
	if err != nil {
		adapter.Logger.Error(err, "ERROR: Failed to delete Escalation policy")
		return err
	}

	adapter.Logger.Info("Policy deleted...")
	return nil
}

func (adapter *EPAdapter) UpdatePDEscalationPolicy(k8sPDPolicy *v1alpha1.EscalationPolicy) error {

	adapter.Logger.Info("Updating policy...")
	_, err := adapter.PD_Client.UpdateEscalationPolicyWithContext(
		context.TODO(),
		k8sPDPolicy.Status.PolicyID,
		adapter.convert(k8sPDPolicy),
	)

	if err != nil {
		adapter.Logger.Error(err, "API Failed to update Escalation policy")
		return err
	}

	adapter.Logger.Info("Upstream Escalation Policy updated...")
	return nil
}

func (adapter *EPAdapter) EqualToUpstream(k8sPolicy v1alpha1.EscalationPolicy) (bool, error) {
	PDPolicy, err := adapter.GetPDEscalationPolicy(k8sPolicy.Status.PolicyID)
	if err != nil {
		adapter.Logger.Error(err, "Failed to get Escalation policy")
		return false, err
	}

	return k8sPolicy.Spec.Name == PDPolicy.Name &&
		k8sPolicy.Spec.Description == PDPolicy.Description &&
		k8sPolicy.Spec.NumLoops == PDPolicy.NumLoops &&
		k8sPolicy.Spec.OnCallHandoffNotifications == PDPolicy.OnCallHandoffNotifications &&
		k8sPolicy.Spec.EscalationRules.CompareAPIObject(PDPolicy.EscalationRules), nil
}

func (adapter *EPAdapter) GetPDEscalationPolicy(id string) (*pagerduty.EscalationPolicy, error) {
	PDPolicy, err := adapter.PD_Client.GetEscalationPolicyWithContext(context.TODO(), id, &pagerduty.GetEscalationPolicyOptions{})
	if err != nil {
		adapter.Logger.Error(err, "Failed to get Escalation policy")
		return nil, err
	}

	adapter.Logger.Info("Escalation policy retrieved", "PDPolicy", PDPolicy)
	return PDPolicy, nil
}
