package discovery

import (
	"github.com/mcorbin/cabourotte/discovery/http"
)

// Configuration the service discovery mechanisms configuration
type Configuration struct {
	HTTP []http.Configuration
}
