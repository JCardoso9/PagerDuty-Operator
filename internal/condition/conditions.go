package condition

import (
	pdv1alpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is a wrapper object for actual Condition functions to allow for easier mocking/testing.
//
//go:generate mockgen -destination=../util/mocks/$GOPACKAGE/conditions.go -package=$GOPACKAGE -source conditions.go
type Conditions interface {
	SetCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType, status metav1.ConditionStatus, reason string, message string)
	FindCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType) (*metav1.Condition, bool)
	HasCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType) bool
}

type ConditionManager struct {
}

// NewConditionManager returns a ConditionManager object
func NewConditionManager() Conditions {
	return &ConditionManager{}
}

// SetCondition sets a condition on a custom resource's status
func (c *ConditionManager) SetCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType, status metav1.ConditionStatus, reason string, message string) {
	now := metav1.Now()
	condition, _ := c.FindCondition(conditions, conditionType)
	if message != condition.Message ||
		status != condition.Status ||
		reason != condition.Reason ||
		conditionType.String() != condition.Type {

		condition.LastTransitionTime = now
	}
	if message != "" {
		condition.Message = message
	}
	condition.Reason = reason
	condition.Status = status
}

// FindCondition finds the suitable Condition object
// by looking for adapter's condition list.
// If none exists, it appends one.
// the second return code is true if the condition already existed before
func (c *ConditionManager) FindCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType) (*metav1.Condition, bool) {
	for i, condition := range *conditions {
		if condition.Type == conditionType.String() {
			return &(*conditions)[i], true
		}
	}

	*conditions = append(
		*conditions,
		metav1.Condition{
			Type: conditionType.String(),
		},
	)

	return &(*conditions)[len(*conditions)-1], false
}

// HasCondition checks for the existance of a given Condition type
func (c *ConditionManager) HasCondition(conditions *[]metav1.Condition, conditionType pdv1alpha1.ConditionType) bool {
	for _, condition := range *conditions {
		if condition.Type == conditionType.String() {
			return true
		}
	}
	return false
}
