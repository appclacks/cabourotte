package discovery

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mcorbin/cabourotte/healthcheck"
)

type HealthcheckSpec struct {
	CommandChecks []healthcheck.CommandHealthcheckConfiguration `json:"command-checks"`
	DNSChecks     []healthcheck.DNSHealthcheckConfiguration     `json:"dns-checks"`
	TCPChecks     []healthcheck.TCPHealthcheckConfiguration     `json:"tcp-checks"`
	HTTPChecks    []healthcheck.HTTPHealthcheckConfiguration    `json:"http-checks"`
	TLSChecks     []healthcheck.TLSHealthcheckConfiguration     `json:"tls-checks"`
}

type HealthcheckStatus struct {
	Created bool `json:"created"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Healthcheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealthcheckSpec   `json:"spec"`
	Status HealthcheckStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HealthcheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Healthcheck `json:"items"`
}
