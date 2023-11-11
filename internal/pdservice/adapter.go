package pdservice

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

type Adapter interface {
	convert(*v1alpha1.PagerdutyServiceSpec) pagerduty.Service
	CreatePDService(k8sPDService *v1alpha1.PagerdutyServiceSpec) (string, error)
	GetService(string) (*pagerduty.Service, error)
	DeleteService(string) error
	UpdateService(*pagerduty.Service) error
	EqualToUpstream(*pagerduty.Service) bool
}

type PDServiceAdapter struct {
	Logger    logr.Logger
	PD_Client *pagerduty.Client
}

func (adapter *PDServiceAdapter) convertSpec(spec *v1alpha1.PagerdutyServiceSpec) pagerduty.Service {
	return pagerduty.Service{
		Name:                   spec.Name,
		Description:            spec.Description,
		AutoResolveTimeout:     spec.AutoResolveTimeout,
		AcknowledgementTimeout: spec.AcknowledgementTimeout,
		Status:                 spec.Status,
		// SupportHours:           spec.SupportHours.Convert(),
		// IncidentUrgencyRule:    spec.IncidentUrgencyRule.Convert(),
		// ScheduledActions:       spec.ScheduledActions.Convert(),
		AlertCreation:    spec.AlertCreation,
		EscalationPolicy: spec.EscalationPolicyID.ToSpecificObject(),
	}
}

func (adapter *PDServiceAdapter) convert(pdService *v1alpha1.PagerdutyService) pagerduty.Service {
	return pagerduty.Service{
		APIObject: pagerduty.APIObject{
			ID:   pdService.Status.ServiceID,
			Type: "service",
		},
		Name:                   pdService.Spec.Name,
		Description:            pdService.Spec.Description,
		AutoResolveTimeout:     pdService.Spec.AutoResolveTimeout,
		AcknowledgementTimeout: pdService.Spec.AcknowledgementTimeout,
		Status:                 pdService.Spec.Status,
		// SupportHours:           pdService.SupportHours.Convert(),
		// IncidentUrgencyRule:    pdService.IncidentUrgencyRule.Convert(),
		// ScheduledActions:       pdService.ScheduledActions.Convert(),
		AlertCreation:    pdService.Spec.AlertCreation,
		EscalationPolicy: pdService.Spec.EscalationPolicyID.ToSpecificObject(),
	}
}

func (adapter *PDServiceAdapter) CreatePDService(k8sPDService *v1alpha1.PagerdutyServiceSpec) (string, error) {
	// Handle Escalation Policy
	// Check if it exists in cluster, if so, add this service
	// If not then create new escalation policy CRD and wait for it to be created
	// Get Escalation Policy ID, update this service with the ID
	// Finally you can create the service

	res, err := adapter.PD_Client.CreateServiceWithContext(context.TODO(), adapter.convertSpec(k8sPDService))
	if err != nil {
		adapter.Logger.Error(err, "PagerDuty Service creation unsuccessfull...")
		return "", err
	}

	return res.ID, nil
}

func (adapter *PDServiceAdapter) GetPDService(id string) (*pagerduty.Service, error) {
	PDService, err := adapter.PD_Client.GetServiceWithContext(context.TODO(), id, &pagerduty.GetServiceOptions{})
	if err != nil {
		adapter.Logger.Error(err, "Failed to get PagerDuty Service")
		return nil, err
	}

	adapter.Logger.Info("PagerDuty Service retrieved", "PDService", PDService)
	return PDService, nil
}

func (adapter *PDServiceAdapter) UpdatePDService(k8sPDService *v1alpha1.PagerdutyService) error {
	adapter.Logger.Info("Updating upstream service with API call...")
	_, err := adapter.PD_Client.UpdateServiceWithContext(context.TODO(), adapter.convert(k8sPDService))

	if err != nil {
		adapter.Logger.Error(err, "API Failed to update PagerDuty Service")
		return err
	}

	adapter.Logger.Info("Upstream PagerDuty Service updated...")
	return nil
}

func (adapter *PDServiceAdapter) DeletePDService(id string) error {
	adapter.Logger.Info("Deleting PagerDuty Service...")

	// TODO: Check if it's necessary to delete the escalation policy

	err := adapter.PD_Client.DeleteServiceWithContext(context.TODO(), id)
	if err != nil {
		adapter.Logger.Error(err, "ERROR: Failed to delete PagerDuty Service")
		return err
	}

	adapter.Logger.Info("PagerDuty Service deleted...")
	return nil
}

func (adapter *PDServiceAdapter) EqualToUpstream(k8sPDService *v1alpha1.PagerdutyService) (bool, error) {
	PDService, err := adapter.GetPDService(k8sPDService.Status.ServiceID)
	if err != nil {
		adapter.Logger.Error(err, "Failed to get Escalation policy")
		return false, err
	}

	convertedk8sPDService := adapter.convert(k8sPDService)
	return convertedk8sPDService.Name == PDService.Name, nil //&&
	// k8sService.Description == upstreamService.Description &&
	// k8sService.AutoResolveTimeout == upstreamService.AutoResolveTimeout &&
	// k8sService.AcknowledgementTimeout == upstreamService.AcknowledgementTimeout &&
	// k8sService.Status == upstreamService.Status &&
	// k8sService.SupportHours == upstreamService.SupportHours &&
	// k8sService.IncidentUrgencyRule == upstreamService.IncidentUrgencyRule &&
	// k8sService.AlertCreation == upstreamService.AlertCreation &&
	// k8sService.EscalationPolicy.ID == upstreamService.EscalationPolicy.ID &&
	// k8sService.AutoPauseNotificationsParameters == upstreamService.AutoPauseNotificationsParameters &&
	// e.scheduledActionsEqual(k8sService, upstreamService.ScheduledActions)

}
