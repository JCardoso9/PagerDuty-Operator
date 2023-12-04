/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PagerdutyServiceSpec defines the desired state of PagerdutyService
type PagerdutyServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Name defines the name of the PagerDuty service that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// Description defines the description of the PagerDuty service that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=""
	Description string `json:"description,omitempty"`

	// Time in seconds that an incident is automatically resolved if left open for that long.
	// Value is null if the feature is disabled. Value must not be negative.
	//Setting this field to 0, null (or unset in POST request) will disable the feature
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=14400
	AutoResolveTimeout *uint `json:"auto_resolve_timeout,omitempty"`

	// Time in seconds that an incident changes to the Triggered State after being Acknowledged.
	// Value is null if the feature is disabled. Value must not be negative.
	// Setting this field to 0, null (or unset in POST request) will disable the feature.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1800
	AcknowledgementTimeout *uint `json:"acknowledgement_timeout,omitempty"`

	// The current state of the Service.
	// +kubebuilder:validation:Enum=active;warning;critical;maintenance;disabled
	// +kubebuilder:default=active
	Status string `json:"status,omitempty"`

	// EscalationPolicyName defines the name of the escalation policy in the cluster that will attributed to the PagerDuty service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	EscalationPolicyName string `json:"escalation_policy_ref,omitempty"`

	// Whether a service creates only incidents, or both alerts and incidents.
	// A service must create alerts in order to enable incident merging.
	// "create_incidents" - The service will create one incident and zero alerts for each incoming event.
	// "create_alerts_and_incidents" - The service will create one incident and one associated alert for each incoming event.
	// +kubebuilder:validation:Enum=create_incidents;create_alerts_and_incidents
	// +kubebuilder:default=create_incidents
	AlertCreation string `json:"alert_creation,omitempty"`
}

// PagerdutyServiceStatus defines the observed state of PagerdutyService
type PagerdutyServiceStatus struct {
	// Represents the observations of a Memcached's current state.
	// Memcached.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Memcached.status.conditions.status are one of True, False, Unknown.
	// Memcached.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Memcached.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// +operator-sdk:csv:customresourcedefinitions:type=status
	// ServiceID stores the ID of the created service
	// +kubebuilder:default=""
	ServiceID string `json:"service_id,omitempty"`

	// EscalationPolicyID stores the ID of the escalation policy that is attributed to the service
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +kubebuilder:default=""
	EscalationPolicyID string `json:"escalation_policy_id,omitempty"`

	// // Conditions store the status conditions of the Service
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PagerdutyService is the Schema for the pagerdutyservices API
type PagerdutyService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PagerdutyServiceSpec   `json:"spec,omitempty"`
	Status PagerdutyServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PagerdutyServiceList contains a list of PagerdutyService
type PagerdutyServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PagerdutyService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PagerdutyService{}, &PagerdutyServiceList{})
}
