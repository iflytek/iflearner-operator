/*
Copyright 2022.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IflearnerRole string

const (
	RoleServer IflearnerRole = "server"
	RoleClient IflearnerRole = "client"
)

// IflearnerJobSpec defines the desired state of IflearnerJob
type IflearnerJobSpec struct {
	SchedulerName string `json:"schedulername,omitempty"`

	Role IflearnerRole `json:"role"`

	Host string `json:"host,omitempty"`

	Template *corev1.PodTemplateSpec `json:"template"`
}

// IflearnerJobStatus defines the observed state of IflearnerJob
type IflearnerJobStatus struct {
	corev1.PodStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IflearnerJob is the Schema for the iflearnerjobs API
type IflearnerJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IflearnerJobSpec   `json:"spec,omitempty"`
	Status IflearnerJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IflearnerJobList contains a list of IflearnerJob
type IflearnerJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IflearnerJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IflearnerJob{}, &IflearnerJobList{})
}
