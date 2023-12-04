package escalation_policy

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

type EPMockAdapter struct {
	Logger logr.Logger
}

var Default_policy_name = "default-policy"
var Default_policy_description = "default_policy_description"
var Default_num_loops uint = 2
var Default_on_call_handoff_notifications = "if_has_services"

var policies map[string]pagerduty.EscalationPolicy = make(map[string]pagerduty.EscalationPolicy)

func (adapter EPMockAdapter) convert(policy *v1alpha1.EscalationPolicy) pagerduty.EscalationPolicy {
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

func (spec EPMockAdapter) convertSpec(policy *v1alpha1.EscalationPolicySpec) pagerduty.EscalationPolicy {
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

func (adapter *EPMockAdapter) CreateEscalationPolicy(k8sPDEscalationPolicy *v1alpha1.EscalationPolicySpec) (string, error) {
	adapter.Logger.Info("Policy created...")
	policies[k8sPDEscalationPolicy.Name] = adapter.convertSpec(k8sPDEscalationPolicy)

	return k8sPDEscalationPolicy.Name, nil
}

func (adapter *EPMockAdapter) DeletePDEscalationPolicy(id string) error {
	delete(policies, id)

	adapter.Logger.Info("Policy deleted...")
	return nil
}

func (adapter *EPMockAdapter) UpdatePDEscalationPolicy(k8sPDPolicy *v1alpha1.EscalationPolicy) error {
	policies[k8sPDPolicy.Status.PolicyID] = adapter.convert(k8sPDPolicy)
	adapter.Logger.Info("Upstream Escalation Policy updated...")
	return nil
}

func (adapter *EPMockAdapter) EqualToUpstream(k8sPolicy v1alpha1.EscalationPolicy) (bool, error) {
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

func (adapter *EPMockAdapter) GetPDEscalationPolicy(id string) (*pagerduty.EscalationPolicy, error) {
	policy, ok := policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found")
	}

	return &policy, nil
}
