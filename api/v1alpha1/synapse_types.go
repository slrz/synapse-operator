/*
Copyright © 2020 The synapse-operator Authors

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

// NOTE: json tags are required.  Any new fields you add must have json tags
// for the fields to be serialized.

// SynapseSpec defines the desired state of Synapse
type SynapseSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// ServerName is a synapse server's public DNS name
	ServerName string `json:"serverName"`

	// ReportStats enables anonymous statistics reporting
	ReportStats bool `json:"reportStats"`

	// Image specifies the container image used for running Synapse.
	// Defaults to "docker.io/matrixdotorg/synapse:latest" if not
	// specified.
	// +optional
	Image string `json:"image,omitempty"`
}

// SynapseStatus defines the observed state of Synapse
type SynapseStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// ConfigMapName is the name of the K8s config map holding the
	// homeserver configuration file(s)
	ConfigMapName string `json:"configMapName,omitempty"`

	// SecretName is the name of the K8s secret storing the server's
	// signing key as well as other secrets used by synapse.
	SecretName string `json:"secretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=synapses
// +kubebuilder:subresource:status

// Synapse is the Schema for the synapses API
type Synapse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SynapseSpec   `json:"spec,omitempty"`
	Status SynapseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SynapseList contains a list of Synapse
type SynapseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Synapse `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Synapse{}, &SynapseList{})
}
