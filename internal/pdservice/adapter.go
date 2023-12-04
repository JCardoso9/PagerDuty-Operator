package pdservice

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/typeinfo"
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

var pdservice_reference_type string = "service"

// func (adapter *PDServiceAdapter) convertSpec(spec *v1alpha1.PagerdutyServiceSpec) pagerduty.Service {
// 	return pagerduty.Service{
// 		Name:                   spec.Name,
// 		Description:            spec.Description,
// 		AutoResolveTimeout:     spec.AutoResolveTimeout,
// 		AcknowledgementTimeout: spec.AcknowledgementTimeout,
// 		Status:                 spec.Status,
// 		AlertCreation:          spec.AlertCreation,
// 		EscalationPolicy:       spec.EscalationPolicyID.ToSpecificObject(),
// 	}
// }

func (adapter *PDServiceAdapter) convert(pdService *v1alpha1.PagerdutyService) pagerduty.Service {
	return pagerduty.Service{
		APIObject: pagerduty.APIObject{
			ID:   pdService.Status.ServiceID,
			Type: pdservice_reference_type,
		},
		Name:                   pdService.Spec.Name,
		Description:            pdService.Spec.Description,
		AutoResolveTimeout:     pdService.Spec.AutoResolveTimeout,
		AcknowledgementTimeout: pdService.Spec.AcknowledgementTimeout,
		Status:                 pdService.Spec.Status,
		AlertCreation:          pdService.Spec.AlertCreation,
		EscalationPolicy:       typeinfo.EscalationPolicyID(pdService.Status.EscalationPolicyID).ToSpecificObject(),
	}
}

func (adapter *PDServiceAdapter) CreatePDService(k8sPDService *v1alpha1.PagerdutyService) (string, error) {
	// Handle Escalation Policy
	// Check if it exists in cluster, if so, add this service
	// If not then create new escalation policy CRD and wait for it to be created
	// Get Escalation Policy ID, update this service with the ID
	// Finally you can create the service

	res, err := adapter.PD_Client.CreateServiceWithContext(context.TODO(), adapter.convert(k8sPDService))
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

	return convertedk8sPDService.Name == PDService.Name &&
		convertedk8sPDService.Description == PDService.Description &&
		*convertedk8sPDService.AutoResolveTimeout == *PDService.AutoResolveTimeout &&
		*convertedk8sPDService.AcknowledgementTimeout == *PDService.AcknowledgementTimeout &&
		convertedk8sPDService.Status == PDService.Status &&
		convertedk8sPDService.AlertCreation == PDService.AlertCreation &&
		convertedk8sPDService.EscalationPolicy.ID == PDService.EscalationPolicy.ID, nil
}
