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

// BusinessServiceSpec defines the desired state of BusinessService
type BusinessServiceSpec struct {

	// Name defines the name of the Business Service that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// Description defines the description of the Business Service that will be created
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=""
	Description string `json:"description,omitempty"`

	// PointOfContact defines the owner of the Business Service.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=""
	PointOfContact string `json:"point_of_contact,omitempty"`

	// TeamID defines the team that owns the Business Service.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=""
	TeamID string `json:"team,omitempty"`
}

// BusinessServiceStatus defines the observed state of BusinessService
type BusinessServiceStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// BusinessServiceID stores the ID of the Business Service
	BusinessServiceID string `json:"business_service_id ,omitempty"`

	//	Conditions stores the conditions of the Business Service
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BusinessService is the Schema for the businessservices API
type BusinessService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessServiceSpec   `json:"spec,omitempty"`
	Status BusinessServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BusinessServiceList contains a list of BusinessService
type BusinessServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessService{}, &BusinessServiceList{})
}
