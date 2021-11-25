package discovery

// KubernetesPod pod discovery
type KubernetesPod struct {
	Labels    map[string]string
	Enabled   bool
	Namespace string
}

// KubernetesService service discovery
type KubernetesService struct {
	Labels    map[string]string
	Enabled   bool
	Namespace string
}

// KubernetesConfiguration Kubernetes service discovery
type KubernetesConfiguration struct {
	Pod     KubernetesPod
	Service KubernetesService
}

// Configuration the service discovery mechanisms configuration
type Configuration struct {
	Kubernetes KubernetesConfiguration
}
