package discovery

import (
	"github.com/appclacks/cabourotte/discovery/http"
)

// Configuration the service discovery mechanisms configuration
type Configuration struct {
	HTTP []http.Configuration
}
