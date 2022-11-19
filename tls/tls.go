package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

// GetTLSConfig returns a tls configuration
func GetTLSConfig(keyPath string, certPath string, cacertPath string, insecure bool) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to load certificates")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if cacertPath != "" {
		caCert, err := os.ReadFile(cacertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to load the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		result := caCertPool.AppendCertsFromPEM(caCert)
		if !result {
			return nil, fmt.Errorf("fail to read ca certificate on %s", certPath)
		}
		tlsConfig.RootCAs = caCertPool

	}
	tlsConfig.InsecureSkipVerify = insecure
	return tlsConfig, nil
}
