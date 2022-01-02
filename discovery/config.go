package discovery

import (
	"github.com/mcorbin/cabourotte/discovery/http"
	"github.com/mcorbin/cabourotte/discovery/kubernetes"
)

// Configuration the service discovery mechanisms configuration
type Configuration struct {
	Kubernetes kubernetes.KubernetesConfiguration
	HTTP       http.Configuration
}
