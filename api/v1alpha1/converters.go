package v1alpha1

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
)

func (spec *PagerdutyServiceSpec) Convert() pagerduty.Service {
	return pagerduty.Service{
		Name:                   spec.Name,
		Description:            spec.Description,
		AutoResolveTimeout:     spec.AutoResolveTimeout,
		AcknowledgementTimeout: spec.AcknowledgementTimeout,
		EscalationPolicy:       spec.EscalationPolicyID.ToSpecificObject(),
	}
}

func (spec *EscalationPolicySpec) Convert() pagerduty.EscalationPolicy {
	if spec.Team == "" {
		fmt.Println("------------------------------------ Team is empty")
		return pagerduty.EscalationPolicy{
			Name:                       spec.Name,
			Description:                spec.Description,
			OnCallHandoffNotifications: spec.OnCallHandoffNotifications,
			NumLoops:                   spec.NumLoops,
			EscalationRules:            spec.EscalationRules.ConvertToPagerDutyObj(),
		}
	}

	return pagerduty.EscalationPolicy{
		Name:                       spec.Name,
		Description:                spec.Description,
		OnCallHandoffNotifications: spec.OnCallHandoffNotifications,
		NumLoops:                   spec.NumLoops,
		Teams:                      []pagerduty.APIReference{spec.Team.ToReference()},
		EscalationRules:            spec.EscalationRules.ConvertToPagerDutyObj(),
	}
}

func convertServiceToAPIStruct(pdService *PagerdutyService) pagerduty.Service {
	return pagerduty.Service{
		Name:                   pdService.Spec.Name,
		Description:            pdService.Spec.Description,
		AutoResolveTimeout:     pdService.Spec.AutoResolveTimeout,
		AcknowledgementTimeout: pdService.Spec.AcknowledgementTimeout,
		EscalationPolicy:       pdService.Spec.EscalationPolicyID.ToSpecificObject(),
	}
}

// deepcopy
