package business_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/condition"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_errors"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const businessServiceFinalizer = "pagerduty.platform.share-now.com/business_service"
const businessServiceReady = "PDBusinessServiceReady"
const RequeWaitTime = time.Second * 10

type SubroutineHandler struct {
	BusinessService  *v1alpha1.BusinessService
	Logger           logr.Logger
	K8sClient        client.Client
	BSAdapter        *BSAdapter
	conditionManager condition.Conditions
}

func (e *SubroutineHandler) ReconcileCreation() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Business Service  Creation...")

	// no upstream policy has been created yet
	if !e.BusinessServiceIDExists() {
		e.Logger.Info("Upstream Business Service not found. Creating...")

		// Handles creation of upstream policy
		businessServiceID, err := e.BSAdapter.CreateBusinessService(&e.BusinessService.Spec)
		if err != nil {
			e.Logger.Error(err, "Failed to create PD Business Service")
			return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, err, err.Error())
		}

		e.Logger.Info("Updating Business Service status...")
		// TODO: Update the rest of the policy status/spec
		e.BusinessService.Status.BusinessServiceID = businessServiceID
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, nil, "Business Service created")
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileDeletion() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Business Service Deletion...")

	if e.deletionTimestampExists() {
		e.Logger.Info("Deletion timestamp for Business Service found. Deleting...")

		if e.BusinessServiceIDExists() {
			e.Logger.Info("Upstream Business Service found, making API deletion call for Business Service ...")
			if err := e.BSAdapter.DeleteBusinessService(e.BusinessService.Status.BusinessServiceID); err != nil {
				e.Logger.Error(err, "Failed to delete Business Service")
				return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, err, err.Error())
			}
		}

		err := e.removeFinalizer()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("No deletion timestamp found for Business Service. Skipping deletion...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) ReconcileUpdate() (pd_utils.OperationResult, error) {
	e.Logger.Info("Reconcile Business Service Update...")

	if !e.BusinessServiceIDExists() {
		e.Logger.Info("No upstream Business Service created yet. Skipping Update...")
		return pd_utils.ContinueProcessing()
	}

	equal, err := e.BSAdapter.EqualToUpstream(*e.BusinessService)
	if err != nil {
		e.Logger.Error(err, "Failed to compare Business Service spec with upstream")
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, err, err.Error())
	}

	if !equal {
		e.Logger.Info("Business Service spec does not match upstream service. Updating...")
		err := e.BSAdapter.UpdateBusinessService(e.BusinessService)

		if err != nil {
			e.Logger.Error(err, "Failed to update Business Service")
			return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, err, err.Error())
		}

		e.Logger.Info("Business Service changed...")
		return e.SetEscalationPolicyCondition(v1alpha1.ConditionReady, businessServiceReady, nil, "PagerDuty Business Service matches upstream service")
	}

	e.Logger.Info("PagerDuty Business Service not changed, Reconcile Update PagerDuty Business Service done...")
	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) BusinessServiceIDExists() bool {
	return e.BusinessService.Status.BusinessServiceID != ""
}

func (e *SubroutineHandler) deletionTimestampExists() bool {
	return !e.BusinessService.GetDeletionTimestamp().IsZero()
}

func (e *SubroutineHandler) AddFinalizer() (pd_utils.OperationResult, error) {
	if !controllerutil.ContainsFinalizer(e.BusinessService, businessServiceFinalizer) {
		e.Logger.Info("Adding Finalizer for Business Service")
		if ok := controllerutil.AddFinalizer(e.BusinessService, businessServiceFinalizer); !ok {
			e.Logger.Info("Could not add finalizer for Business Service")
			return pd_utils.Requeue()
		}

		err := e.K8sClient.Update(context.Background(), e.BusinessService)
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	return pd_utils.ContinueProcessing()
}

func (e *SubroutineHandler) removeFinalizer() error {
	if !controllerutil.ContainsFinalizer(e.BusinessService, businessServiceFinalizer) {
		e.Logger.Info("No Finalizer present in Business Service, skipping finalizer removal...")
		return nil
	}

	e.Logger.Info("Removing Finalizer for Business Service after successfully perform the operations")
	if ok := controllerutil.RemoveFinalizer(e.BusinessService, businessServiceFinalizer); !ok {
		e.Logger.Info("Failed to remove finalizer for Business Service")
		return errors.New("failed to remove finalizer for Business Service")
	}

	return e.K8sClient.Update(context.TODO(), e.BusinessService)
}

func (e *SubroutineHandler) StatusUpdate() error {
	e.Logger.Info("Updating status...")
	if err := e.K8sClient.Status().Update(context.TODO(), e.BusinessService); err != nil {
		e.Logger.Error(err, "Failed to update Business Service status")
		return pd_errors.Wrap(err, fmt.Sprintf("failed to update Business Service state for %s", e.BusinessService.Name))
	}

	e.Logger.Info("Status updated...")
	return nil
}

func (e *SubroutineHandler) SetEscalationPolicyCondition(conditionType v1alpha1.ConditionType, reason string, err error, message string) (pd_utils.OperationResult, error) {
	conditions := &e.BusinessService.Status.Conditions

	// defer e.StatusUpdate()??

	if err != nil {
		e.Logger.Info("Setting ready condition to false", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message, "error", err.Error())
		e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionFalse, reason, err.Error())

		err := e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update Business Service to false ready condition, Requeing in 10 seconds... ")
		}

		return pd_utils.RequeueAfter(RequeWaitTime, err)
	}

	// TODO: Can probably simplify this if by only setting condition if on of these conditions is false...
	if len(*conditions) > 0 && (*conditions)[0].Status == metav1.ConditionTrue && (*conditions)[0].Message == message {
		err = e.StatusUpdate()
		if err != nil {
			e.Logger.Error(err, "Failed to update Business Service to true ready condition ")
			return pd_utils.RequeueAfter(RequeWaitTime, err)
		}
		return pd_utils.StopProcessing()
	}

	e.Logger.Info("Setting ready condition to true", "conditionType", conditionType, "status", metav1.ConditionFalse, "reason", reason, "message", message)
	e.conditionManager.SetCondition(conditions, conditionType, metav1.ConditionTrue, reason, message)
	err = e.StatusUpdate()

	return pd_utils.RequeueOnErrorOrStop(err)
}

func (e *SubroutineHandler) Initialization() (pd_utils.OperationResult, error) {
	e.Logger.Info("Starting Initialization...")
	if e.BusinessService.Status.Conditions == nil {
		e.BusinessService.Status.Conditions = []metav1.Condition{}

		err := e.StatusUpdate()
		return pd_utils.RequeueOnErrorOrStop(err)
	}

	e.Logger.Info("Initialization done...")
	return pd_utils.ContinueProcessing()
}
