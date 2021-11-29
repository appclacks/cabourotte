package kubernetes

// KubernetesPod pod discovery
type KubernetesPod struct {
	Labels    map[string]string
	Enabled   bool
	Namespace string
}

// KubernetesPod pod discovery
type KubernetesCRD struct {
	Enabled   bool
	Namespace string
	Labels    map[string]string
}

// KubernetesService service discovery
type KubernetesService struct {
	Labels    map[string]string
	Enabled   bool
	Namespace string
}

// KubernetesConfiguration Kubernetes service discovery
type KubernetesConfiguration struct {
	CRD     KubernetesCRD
	Pod     KubernetesPod
	Service KubernetesService
}
