package typeinfo

import (
	"github.com/PagerDuty/go-pagerduty"
)

type K8sEscalationRule struct {
	Delay   uint       `json:"escalation_delay_in_minutes,omitempty"`
	Targets UserIDList `json:"targets"`
}

type K8sEscalationRuleList []K8sEscalationRule

func (rule *K8sEscalationRule) ConvertToPagerDutyObj() pagerduty.EscalationRule {
	return pagerduty.EscalationRule{
		Delay:   rule.Delay,
		Targets: rule.Targets.ToAPIObject(),
	}
}

func (rules K8sEscalationRuleList) ConvertToPagerDutyObj() []pagerduty.EscalationRule {
	var pdRules []pagerduty.EscalationRule
	for _, rule := range rules {
		pdRules = append(pdRules, rule.ConvertToPagerDutyObj())
	}
	return pdRules
}

func (rule *K8sEscalationRule) CompareAPIObject(apiObject pagerduty.EscalationRule) bool {
	return rule.Delay == apiObject.Delay &&
		rule.Targets.compareAPIObject(apiObject.Targets)
}

func (rule K8sEscalationRuleList) CompareAPIObject(apiObject []pagerduty.EscalationRule) bool {
	if len(rule) != len(apiObject) {
		return false
	}
	for i, r := range rule {
		if !r.CompareAPIObject(apiObject[i]) {
			return false
		}
	}
	return true
}

// Deepcopy
func (in *K8sEscalationRule) DeepCopyInto(out *K8sEscalationRule) {
	*out = *in
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make(UserIDList, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy
func (in *K8sEscalationRule) DeepCopy() *K8sEscalationRule {
	if in == nil {
		return nil
	}
	out := new(K8sEscalationRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject
func (in *K8sEscalationRule) DeepCopyObject() interface{} {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
