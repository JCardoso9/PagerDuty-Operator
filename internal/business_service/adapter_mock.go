package business_service

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

type BSMockAdapter struct {
	Logger logr.Logger
}

var Default_busService_name = "default-busService"
var Default_busService_description = "default_busService_description"
var Default_busService_pointOfContact = "TestUser"
var Default_busService_teamID = "MOCKTEAMID"

var busServices map[string]pagerduty.BusinessService = make(map[string]pagerduty.BusinessService)

func (adapter *BSMockAdapter) convert(bsService *v1alpha1.BusinessService) *pagerduty.BusinessService {
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

func (spec *BSMockAdapter) convertSpec(bsService *v1alpha1.BusinessServiceSpec) *pagerduty.BusinessService {
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

func (adapter *BSMockAdapter) CreateBusinessService(k8sPDBusinessServiceSpec *v1alpha1.BusinessServiceSpec) (string, error) {
	adapter.Logger.Info("busService created...")
	busServices[k8sPDBusinessServiceSpec.Name] = *adapter.convertSpec(k8sPDBusinessServiceSpec)

	return k8sPDBusinessServiceSpec.Name, nil
}

func (adapter *BSMockAdapter) DeleteBusinessService(id string) error {
	delete(busServices, id)

	adapter.Logger.Info("busService deleted...")
	return nil
}

func (adapter *BSMockAdapter) UpdateBusinessService(k8sPDBusinessService *v1alpha1.BusinessService) error {
	busServices[k8sPDBusinessService.Status.BusinessServiceID] = *adapter.convert(k8sPDBusinessService)
	adapter.Logger.Info("Upstream Escalation busService updated...")
	return nil
}

func (adapter *BSMockAdapter) EqualToUpstream(k8sBusinessService v1alpha1.BusinessService) (bool, error) {
	PDPolicy, err := adapter.GetBusinessService(k8sBusinessService.Status.BusinessServiceID)
	if err != nil {
		adapter.Logger.Error(err, "Failed to get Escalation busService")
		return false, err
	}

	return k8sBusinessService.Spec.Name == PDPolicy.Name &&
		k8sBusinessService.Spec.Description == PDPolicy.Description &&
		k8sBusinessService.Spec.PointOfContact == PDPolicy.PointOfContact &&
		k8sBusinessService.Spec.TeamID == PDPolicy.Team.ID, nil
}

func (adapter *BSMockAdapter) GetBusinessService(id string) (*pagerduty.BusinessService, error) {
	busService, ok := busServices[id]
	if !ok {
		return nil, fmt.Errorf("busService not found")
	}

	return &busService, nil
}
