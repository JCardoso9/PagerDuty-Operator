package pdservice

//WIP

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	pdv1alpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_errors"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const pdServiceFinalizer = "pagerduty.platform.share-now.com/service"
const pdServiceReady = "PDServiceReady"

type PDServiceAdapter struct {
	PagerdutyService *v1alpha1.PagerdutyService
	Logger           logr.Logger
	K8sClient        client.Client
	PD_Client        *pagerduty.Client
	conditionManager condition.Conditions
}

func (e *PDServiceAdapter) ReconcileCreation() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Creation...")

	// no upstream policy has been created yet
	if !e.serviceIDExists() {
		e.Logger.Info("Upstream PagerDuty Service not found. Creating...")

		// Handles creation of upstream policy
		serviceID, err := e.createPDService()
		if err != nil {
			e.Logger.Error(err, "Failed to create PagerDuty Service")
			return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, "")
		}

		e.Logger.Info("Updating PagerDuty Service status...")
		// TODO: Update the rest of the policy status/spec
		e.PagerdutyService.Status.ServiceID = serviceID
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, nil, "PagerDuty Service created")
	}

	return pd_utils.ContinueProcessing()
}

func (e *PDServiceAdapter) createPDService() (string, error) {
	if e.PagerdutyService.Status.ServiceID != "" {
		e.Logger.Info("PagerDuty Service already exists. Skipping  creation...")
		return e.PagerdutyService.Status.ServiceID, nil
	}

	// Handle Escalation Policy
	// Check if it exists in cluster, if so, add this service
	// If not then create new escalation policy CRD and wait for it to be created
	// Get Escalation Policy ID, update this service with the ID
	// Finally you can create the service

	res, err := e.PD_Client.CreateServiceWithContext(context.TODO(), e.PagerdutyService.Spec.Convert())
	if err != nil {
		e.Logger.Error(err, "PagerDuty Service creation unsuccessfull...")
		return "", err
	}

	return res.ID, nil
}

func (e *PDServiceAdapter) ReconcileDeletion() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Deletion...")

	if e.deletionTimestampExists() {
		e.Logger.Info("Deletion timestamp found. Deleting...")

		if e.serviceIDExists() {
			e.Logger.Info("Upstream PagerDuty Service found, making API deletion call...")
			if err := e.deletePDService(); err != nil {
				e.Logger.Error(err, "Failed to delete PagerDuty Service")
				return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, "")
			}
		}

		err := e.removeFinalizer()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("No deletion timestamp found. Skipping PagerDuty Service deletion...")
	return pd_utils.ContinueProcessing()
}

func (e *PDServiceAdapter) deletePDService() error {
	e.Logger.Info("Deleting PagerDuty Service...")

	// TODO: Check if it's necessary to delete the escalation policy

	err := e.PD_Client.DeleteServiceWithContext(context.TODO(), e.PagerdutyService.Status.ServiceID)
	if err != nil {
		e.Logger.Error(err, "ERROR: Failed to delete PagerDuty Service")
		return err
	}

	e.Logger.Info("PagerDuty Service deleted...")
	return nil
}

func (e *PDServiceAdapter) ReconcileUpdate() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Update...")

	if !e.serviceIDExists() {
		e.Logger.Info("No upstream pagerduty service created yet. Skipping Update...")
		return pd_utils.ContinueProcessing()
	}

	changed, err := e.updatePDService()
	if err != nil {
		e.Logger.Error(err, "Failed to update PagerDuty Service")
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, "")
	}

	e.Logger.Info("Reconcile Update PagerDuty Service done...")
	if changed {
		e.Logger.Info("PagerDuty Service changed...")
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, nil, "PagerDuty Service matches upstream service")
	}

	e.Logger.Info("PagerDuty Service not changed...")
	return pd_utils.ContinueProcessing()
}

func (e *PDServiceAdapter) updatePDService() (bool, error) {
	PDService, err := e.getPDService(e.PagerdutyService.Status.ServiceID)
	if err != nil {
		e.Logger.Error(err, "Failed to get Escalation policy")
		return false, err
	}

	k8sPDService := e.PagerdutyService.Spec.Convert()
	k8sPDService.ID = e.PagerdutyService.Status.ServiceID

	e.Logger.Info("Comparing PagerDuty Service spec with upstream service...")
	if !e.serviceEqualUpstream(PDService, k8sPDService) {
		e.Logger.Info("Updating service...")
		_, err := e.PD_Client.UpdateServiceWithContext(
			context.TODO(),
			k8sPDService,
		)

		if err != nil {
			e.Logger.Error(err, "API Failed to update PagerDuty Service")
			return false, err
		}

		e.Logger.Info("PagerDuty Service updated...")
		return true, nil
		// Update condition, call status update
	}

	e.Logger.Info("CRD of PagerDuty Service matches upstream...")
	return false, nil
}

func (e *PDServiceAdapter) serviceEqualUpstream(upstreamPolicy *pagerduty.Service, k8sPolicy pagerduty.Service) bool {
	panic("implement me")
}

func (e *PDServiceAdapter) getPDService(id string) (*pagerduty.Service, error) {
	PDService, err := e.PD_Client.GetServiceWithContext(context.TODO(), id, &pagerduty.GetServiceOptions{})
	if err != nil {
		e.Logger.Error(err, "Failed to get PagerDuty Service")
		return nil, err
	}

	e.Logger.Info("PagerDuty Service retrieved", "PDService", PDService)
	return PDService, nil
}

func (e *PDServiceAdapter) serviceIDExists() bool {
	return e.PagerdutyService.Status.ServiceID != ""
}

func (e *PDServiceAdapter) deletionTimestampExists() bool {
	return !e.PagerdutyService.GetDeletionTimestamp().IsZero()
}

func (e *PDServiceAdapter) AddFinalizer() (pd_utils.OperationResult, error) {
	if !controllerutil.ContainsFinalizer(e.PagerdutyService, pdServiceFinalizer) {
		e.Logger.Info("Adding Finalizer for PagerDuty Service")
		if ok := controllerutil.AddFinalizer(e.PagerdutyService, pdServiceFinalizer); !ok {
			e.Logger.Info("Could not add finalizer for PagerDuty Service")
			return pd_utils.Requeue()
		}

		err := e.K8sClient.Update(context.Background(), e.PagerdutyService)
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	return pd_utils.ContinueProcessing()
}

func (e *PDServiceAdapter) removeFinalizer() error {
	if !controllerutil.ContainsFinalizer(e.PagerdutyService, pdServiceFinalizer) {
		e.Logger.Info("No Finalizer present, skipping finalizer removal in PagerDuty Service...")
		return nil
	}

	e.Logger.Info("Removing Finalizer for PagerDuty Service after successfully perform the operations")
	if ok := controllerutil.RemoveFinalizer(e.PagerdutyService, pdServiceFinalizer); !ok {
		e.Logger.Info("Failed to remove finalizer for PagerDuty Service")
		return errors.New("Failed to remove finalizer for PagerDuty Service")
	}

	return e.K8sClient.Update(context.TODO(), e.PagerdutyService)
}

// StatusUpdate updates the project claim status
func (e *PDServiceAdapter) StatusUpdate() error {
	e.Logger.Info("Updating PagerDuty Service status...")
	if err := e.K8sClient.Status().Update(context.TODO(), e.PagerdutyService); err != nil {
		e.Logger.Error(err, "Failed to update PagerDuty Service status")
		return pd_errors.Wrap(err, fmt.Sprintf("failed to update PagerDuty Service state for %s", e.PagerdutyService.Name))
	}

	e.Logger.Info("Status of PagerDuty Service updated...")
	return nil
}

func (e *PDServiceAdapter) SetPagerDutyServiceCondition(conditionType pdv1alpha1.ConditionType, reason string, err error, message string) (pd_utils.OperationResult, error) {
	conditions := &e.PagerdutyService.Status.Conditions

	if err != nil {
		e.Logger.Info("Setting PagerDuty Service's ready condition to false", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message, "error", err.Error())
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, err.Error())

		err := e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update PagerDuty Service to false ready condition ")
		}

		return pd_utils.RequeueAfter(time.Second*10, err)
	}

	if len(*conditions) > 0 && (*conditions)[0].Status == metav1.ConditionTrue && (*conditions)[0].Message == message {
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, message)
		return pd_utils.StopProcessing()
	}

	e.Logger.Info("Setting PagerDuty Service's ready condition to true", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message)
	e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionTrue, reason, message)
	err = e.StatusUpdate()

	return pd_utils.RequeueOnErrorOrStop(err)
}

func (e *PDServiceAdapter) Initialization() (pd_utils.OperationResult, error) {
	e.Logger.Info("Starting Initialization...")
	if e.PagerdutyService.Status.Conditions == nil {
		e.PagerdutyService.Status.Conditions = []metav1.Condition{}

		err := e.StatusUpdate()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("Initialization done...")
	return pd_utils.ContinueProcessing()
}
