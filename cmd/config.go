package cmd

import (
	"github.com/pkg/errors"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/http"
)

// Instance configuration of a Cabourotte instance
type Instance struct {
	Host      string
	Port      uint32
	Path      string
	Protocol  healthcheck.Protocol
	Key       string
	Cert      string
	Cacert    string
	BasicAuth http.BasicAuth `yaml:"basic-auth" json:"basic-auth"`
	Insecure  bool
}

// MainConfiguration The CLI configuration
type MainConfiguration struct {
	defaultInstance string `yaml:"default-instance" json:"default-instance"`
	Instances       map[string]Instance
}
