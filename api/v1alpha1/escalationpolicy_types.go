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
	"gitlab.share-now.com/platform/pagerduty-operator/internal/typeinfo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EscalationRule is a rule for an escalation policy to trigger.
// type EscalationRule struct {
// 	ID    string `json:"id,omitempty"`
// 	Delay uint   `json:"escalation_delay_in_minutes,omitempty"`

// 	// The targets an incident should be assigned to upon reaching this rule.
// 	// +kubebuilder:validation:MinItems=1
// 	// +kubebuilder:validation:MaxItems=10
// 	Targets []typeinfo.APIReference `json:"targets"`
// }

// EscalationPolicySpec defines the desired state of EscalationPolicy
type EscalationPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name defines the name of the Escalation Policy that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// Description defines the description of the Escalation Policy that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=""
	Description string `json:"description,omitempty"`

	// Determines how on call handoff notifications will be sent for users on the escalation policy.
	// Defaults to "if_has_services".
	// +kubebuilder:default=if_has_services
	// +kubebuilder:validation:Enum=if_has_services;always
	OnCallHandoffNotifications string `json:"on_call_handoff_notifications,omitempty"`

	// NumLoops defines he number of times the escalation policy will repeat after reaching the end of its escalation.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	NumLoops uint `json:"num_loops,omitempty"`

	// EscalationRules defines the rules of the Escalation Policy
	// +kubebuilder:validation:Required
	EscalationRules typeinfo.K8sEscalationRuleList `json:"escalation_rules,omitempty"`

	// Team associated with the policy. Account must have the teams ability to use this parameter.
	// Only one team may be associated with the policy.
	// +kubebuilder:default=""
	Team typeinfo.TeamID `json:"teams,omitempty"`
}

// EscalationPolicyStatus defines the observed state of EscalationPolicy
type EscalationPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +operator-sdk:csv:customresourcedefinitions:type=status
	// PolicyID stores the ID of the Escalation Policy
	PolicyID string `json:"policy_id,omitempty"`

	//	Conditions stores the conditions of the Escalation Policy
	// +kubebuilder:default={}
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EscalationPolicy is the Schema for the escalationpolicies API
type EscalationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EscalationPolicySpec   `json:"spec,omitempty"`
	Status EscalationPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EscalationPolicyList contains a list of EscalationPolicy
type EscalationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EscalationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EscalationPolicy{}, &EscalationPolicyList{})
}
