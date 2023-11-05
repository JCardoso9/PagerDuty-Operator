package v1alpha1

// ConditionType is a valid value for Condition.Type
type ConditionType string

const (
	// ConditionReady is set when a pagerduty custom resource state changes to Ready state
	ConditionReady ConditionType = "Ready"
	// ConditionPending is set when a pagerduty custom resource is still waiting for the API calls to finish
	ConditionPending ConditionType = "Pending"
	// ConditionError is set when a pagerduty custom resource is unable to complete the API calls
	ConditionError ConditionType = "Error"
)

func (c ConditionType) String() string {
	return string(c)
}
