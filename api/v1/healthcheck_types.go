/*
Copyright 2021.

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

	"github.com/mcorbin/cabourotte/healthcheck"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HealthcheckSpec defines the desired state of Healthcheck
type HealthcheckSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// CommandChecks healthchecks of type commands
	CommandChecks []healthcheck.CommandHealthcheckConfiguration `yaml:"command-checks" json:"command-checks"`
	// +kubebuilder:validation:Optional
	// DNSChecks healthchecks of type DNS
	DNSChecks []healthcheck.DNSHealthcheckConfiguration `yaml:"dns-checks" json:"dns-checks"`
	// +kubebuilder:validation:Optional
	// TCPChecks healthchecks of type TCP
	TCPChecks []healthcheck.TCPHealthcheckConfiguration `yaml:"tcp-checks" json:"tcp-checks"`
	// +kubebuilder:validation:Optional
	// HTTPChecks healthchecks of type HTTP
	HTTPChecks []healthcheck.HTTPHealthcheckConfiguration `yaml:"http-checks" json:"http-checks"`
	// +kubebuilder:validation:Optional
	//  healthchecks of type TLS
	TLSChecks []healthcheck.TLSHealthcheckConfiguration `yaml:"tls-checks" json:"tls-checks"`
}

// HealthcheckStatus defines the observed state of Healthcheck
type HealthcheckStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Healthcheck is the Schema for the healthchecks API
type Healthcheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealthcheckSpec   `json:"spec,omitempty"`
	Status HealthcheckStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HealthcheckList contains a list of Healthcheck
type HealthcheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Healthcheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Healthcheck{}, &HealthcheckList{})
}
