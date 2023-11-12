package business_service

import (
	"context"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

type Adapter interface {
	convert(*v1alpha1.BusinessServiceSpec) pagerduty.BusinessService
	CreateBusinessService(k8sPDEscalationPolicy *v1alpha1.BusinessServiceSpec) (string, error)
	GetBusinessService(string) (*pagerduty.BusinessService, error)
	DeleteBusinessService(string) error
	UpdateBusinessService(*v1alpha1.BusinessService) error
	EqualToUpstream(v1alpha1.BusinessService) (bool, error)
}

type BSAdapter struct {
	Logger    logr.Logger
	PD_Client *pagerduty.Client
}

var business_service_reference_type = "business_service"

func (adapter *BSAdapter) convert(bsService *v1alpha1.BusinessService) *pagerduty.BusinessService {
	if bsService.Spec.TeamID == "" {
		fmt.Println("------------------------------------ BS Team is empty")
		return &pagerduty.BusinessService{
			ID:             bsService.Status.BusinessServiceID,
			Name:           bsService.Spec.Name,
			Description:    bsService.Spec.Description,
			PointOfContact: bsService.Spec.PointOfContact,
		}
	}

	return &pagerduty.BusinessService{
		ID:             bsService.Status.BusinessServiceID,
		Name:           bsService.Spec.Name,
		Description:    bsService.Spec.Description,
		PointOfContact: bsService.Spec.PointOfContact,
		Team: &pagerduty.BusinessServiceTeam{
			ID:   bsService.Spec.TeamID,
			Type: business_service_reference_type,
		},
	}
}

func (spec *BSAdapter) convertSpec(bsService *v1alpha1.BusinessServiceSpec) *pagerduty.BusinessService {
	if bsService.TeamID == "" {
		fmt.Println("------------------------------------ BS Team is empty")
		return &pagerduty.BusinessService{
			Name:           bsService.Name,
			Description:    bsService.Description,
			PointOfContact: bsService.PointOfContact,
		}
	}

	return &pagerduty.BusinessService{
		Name:           bsService.Name,
		Description:    bsService.Description,
		PointOfContact: bsService.PointOfContact,
		Team: &pagerduty.BusinessServiceTeam{
			ID:   bsService.TeamID,
			Type: business_service_reference_type,
		},
	}
}

func (adapter *BSAdapter) CreateBusinessService(k8sBusinessService *v1alpha1.BusinessServiceSpec) (string, error) {

	res, err := adapter.PD_Client.CreateBusinessServiceWithContext(context.TODO(), adapter.convertSpec(k8sBusinessService))
	if err != nil {
		adapter.Logger.Error(err, "Business Service creation unsuccessfull...")
		return "", err
	}

	return res.ID, nil
}

func (adapter *BSAdapter) DeleteBusinessService(id string) error {
	adapter.Logger.Info("Deleting policy...")

	err := adapter.PD_Client.DeleteBusinessServiceWithContext(context.TODO(), id)
	if err != nil {
		adapter.Logger.Error(err, "ERROR: Failed to delete Business Service")
		return err
	}

	adapter.Logger.Info("Policy deleted...")
	return nil
}

func (adapter *BSAdapter) UpdateBusinessService(k8sBusinessService *v1alpha1.BusinessService) error {

	adapter.Logger.Info("Updating Business Service...")
	_, err := adapter.PD_Client.UpdateBusinessServiceWithContext(
		context.TODO(),
		adapter.convert(k8sBusinessService),
	)

	if err != nil {
		adapter.Logger.Error(err, "API Failed to update Business Service")
		return err
	}

	adapter.Logger.Info("Upstream Business Service updated...")
	return nil
}

func (adapter *BSAdapter) EqualToUpstream(k8sBusinessService v1alpha1.BusinessService) (bool, error) {
	businessService, err := adapter.GetBusinessService(k8sBusinessService.Status.BusinessServiceID)
	if err != nil {
		adapter.Logger.Error(err, "Failed to get upstream Business Service")
		return false, err
	}

	return k8sBusinessService.Spec.Name == businessService.Name &&
		k8sBusinessService.Spec.Description == businessService.Description &&
		k8sBusinessService.Spec.PointOfContact == businessService.PointOfContact &&
		k8sBusinessService.Spec.TeamID == businessService.Team.ID, nil
}

func (adapter *BSAdapter) GetBusinessService(id string) (*pagerduty.BusinessService, error) {
	businessService, err := adapter.PD_Client.GetBusinessServiceWithContext(context.TODO(), id)
	if err != nil {
		adapter.Logger.Error(err, "Failed to get Business Service")
		return nil, err
	}

	adapter.Logger.Info("Business Service retrieved", "businessService", businessService)
	return businessService, nil
}
