/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LinodeMachineTemplateSpec defines the desired state of LinodeMachineTemplate.
type LinodeMachineTemplateSpec struct {
	Template LinodeMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=linodemachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// LinodeMachineTemplate is the Schema for the linodemachinetemplates API.
type LinodeMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LinodeMachineTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// LinodeMachineTemplateList contains a list of LinodeMachineTemplate.
type LinodeMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LinodeMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LinodeMachineTemplate{}, &LinodeMachineTemplateList{})
}
