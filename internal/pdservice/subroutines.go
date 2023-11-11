package pdservice

//WIP

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type SubroutineHandler struct {
	PagerdutyService *v1alpha1.PagerdutyService
	Logger           logr.Logger
	K8sClient        client.Client
	PDServiceAdapter *PDServiceAdapter
	conditionManager condition.Conditions
}

func (e *SubroutineHandler) ReconcileCreation() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Creation...")

	// no upstream policy has been created yet
	if !e.serviceIDExists() {
		e.Logger.Info("Upstream PagerDuty Service not found. Creating...")

		// Handles creation of upstream policy
		serviceID, err := e.PDServiceAdapter.CreatePDService(&e.PagerdutyService.Spec)
		if err != nil {
			e.Logger.Error(err, "Failed to create PagerDuty Service")
			return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, err.Error())
		}

		e.Logger.Info("Updating PagerDuty Service status...")
		// TODO: Update the rest of the policy status/spec
		e.PagerdutyService.Status.ServiceID = serviceID
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, nil, "PagerDuty Service created")
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileDeletion() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Deletion...")

	if e.deletionTimestampExists() {
		e.Logger.Info("Deletion timestamp found. Deleting...")

		if e.serviceIDExists() {
			e.Logger.Info("Upstream PagerDuty Service found, making API deletion call...")
			if err := e.PDServiceAdapter.DeletePDService(e.PagerdutyService.Status.ServiceID); err != nil {
				e.Logger.Error(err, "Failed to delete PagerDuty Service")
				return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, err.Error())
			}
		}

		err := e.removeFinalizer()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("No deletion timestamp found. Skipping PagerDuty Service deletion...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileUpdate() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile PagerDuty Service Update...")

	if !e.serviceIDExists() {
		e.Logger.Info("No upstream pagerduty service created yet. Skipping Update...")
		return pd_utils.ContinueProcessing()
	}

	equal, err := e.PDServiceAdapter.EqualToUpstream(e.PagerdutyService)
	if err != nil {
		e.Logger.Error(err, "Failed to compare PagerDuty Service spec with upstream service")
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, err.Error())
	}

	if !equal {
		e.Logger.Info("PagerDuty Service spec does not match upstream service. Updating...")
		err := e.PDServiceAdapter.UpdatePDService(e.PagerdutyService)

		if err != nil {
			e.Logger.Error(err, "Failed to update PagerDuty Service")
			return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, err, err.Error())
		}

		e.Logger.Info("PagerDuty Service changed...")
		return e.SetPagerDutyServiceCondition(pdv1alpha1.ConditionReady, pdServiceReady, nil, "PagerDuty Service matches upstream service")
	}

	e.Logger.Info("PagerDuty Service not changed, Reconcile Update PagerDuty Service done...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) serviceIDExists() bool {
	return e.PagerdutyService.Status.ServiceID != ""
}

func (e *SubroutineHandler) deletionTimestampExists() bool {
	return !e.PagerdutyService.GetDeletionTimestamp().IsZero()
}

func (e *SubroutineHandler) AddFinalizer() (pd_utils.OperationResult, error) {
	if !controllerutil.ContainsFinalizer(e.PagerdutyService, pdServiceFinalizer) {
		e.Logger.Info("Adding Finalizer for PagerDuty Service")
		if ok := controllerutil.AddFinalizer(e.PagerdutyService, pdServiceFinalizer); !ok {
			e.Logger.Info("Could not add finalizer for PagerDuty Service")
			return pd_utils.Requeue()
		}

		e.Logger.Info("Finalizer was added to pagerdutyservice")
		err := e.K8sClient.Update(context.Background(), e.PagerdutyService)
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) removeFinalizer() error {
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
func (e *SubroutineHandler) StatusUpdate() error {
	e.Logger.Info("Updating PagerDuty Service status...")
	if err := e.K8sClient.Status().Update(context.TODO(), e.PagerdutyService); err != nil {
		e.Logger.Error(err, "Failed to update PagerDuty Service status")
		return pd_errors.Wrap(err, fmt.Sprintf("failed to update PagerDuty Service state for %s", e.PagerdutyService.Name))
	}

	e.Logger.Info("Status of PagerDuty Service updated...")
	return nil
}

func (e *SubroutineHandler) SetPagerDutyServiceCondition(conditionType pdv1alpha1.ConditionType, reason string, err error, message string) (pd_utils.OperationResult, error) {
	conditions := &e.PagerdutyService.Status.Conditions

	// defer e.StatusUpdate()

	if err != nil {
		e.Logger.Info("Setting PagerDuty Service's ready condition to false", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message, "error", err.Error())
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, message)

		err := e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update PagerDuty Service to false ready condition ")
		}

		return pd_utils.RequeueAfter(time.Second*10, err)
	}

	// Same condition as before, stop processing
	if len(*conditions) > 0 && (*conditions)[0].Status == metav1.ConditionTrue && (*conditions)[0].Message == message {
		err = e.StatusUpdate()
		return pd_utils.StopProcessing()
	}

	e.Logger.Info("Setting PagerDuty Service's ready condition to true", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message)
	e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionTrue, reason, message)
	err = e.StatusUpdate()

	return pd_utils.RequeueOnErrorOrStop(err)
}

func (e *SubroutineHandler) Initialization() (pd_utils.OperationResult, error) {
	e.Logger.Info("Starting Initialization...")
	if e.PagerdutyService.Status.Conditions == nil {
		e.PagerdutyService.Status.Conditions = []metav1.Condition{}

		err := e.StatusUpdate()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("Initialization done...")
	return pd_utils.ContinueProcessing()
}
