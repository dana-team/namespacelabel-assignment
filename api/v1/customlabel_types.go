/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomLabelSpec defines the desired state of CustomLabel
type CustomLabelSpec struct {
	//CustomLabels map of custom labels that are added to namespace
	CustomLabels map[string]string `json:"customLabels,omitempty"`
}
type LabelStatus struct {
	Applied bool   `json:"applied"`
	Value   string `json:"value"`
}

// CustomLabelStatus defines the observed state of CustomLabel
type CustomLabelStatus struct {

	//Message Gives additional info regarding the customlabel status
	//or any error that occurred
	Message string `json:"message,omitempty"`

	//PerLabelStatus Information regarding each label in the spec,
	// whether it was applied and the previous value
	PerLabelStatus map[string]LabelStatus `json:"perLabelStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Applied",type=boolean,JSONPath=`.status.applied`

type CustomLabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomLabelSpec   `json:"spec,omitempty"`
	Status CustomLabelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CustomLabelList contains a list of CustomLabels
type CustomLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomLabel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomLabel{}, &CustomLabelList{})
}
