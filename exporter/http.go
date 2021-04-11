package exporter

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

// HTTPExporter the http exporter struct
type HTTPExporter struct {
	Logger *zap.Logger
	URL    string
	Config *HTTPConfiguration
	Client *http.Client
}

// NewHTTPExporter creates a new HTTP exporter
func NewHTTPExporter(logger *zap.Logger, config *HTTPConfiguration) (*HTTPExporter, error) {
	protocol := "http"
	tlsConfig := &tls.Config{}
	if config.Protocol == healthcheck.HTTPS {
		protocol = "https"
	}
	url := fmt.Sprintf(
		"%s://%s%s",
		protocol,
		net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
		config.Path)
	if config.Key != "" {
		cert, err := tls.LoadX509KeyPair(config.Cert, config.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to load certificates")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if config.Cacert != "" {
		caCert, err := ioutil.ReadFile(config.Cacert)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to load the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		result := caCertPool.AppendCertsFromPEM(caCert)
		if !result {
			return nil, fmt.Errorf("fail to read ca certificate for exporter %s", config.Name)
		}
		tlsConfig.RootCAs = caCertPool

	}
	tlsConfig.InsecureSkipVerify = config.Insecure
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	exporter := HTTPExporter{
		Logger: logger,
		Config: config,
		URL:    url,
		Client: &http.Client{
			Transport: transport,
			Timeout:   time.Second * 3,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return &exporter, nil
}

// Start starts the HTTP exporter component
func (c *HTTPExporter) Start() error {
	// nothing to do
	c.Logger.Info(fmt.Sprintf("Starting the HTTP healcheck exporter on %s:%d", c.Config.Host, c.Config.Port))
	return nil
}

// Stop stops the HTTP exporter component
func (c *HTTPExporter) Stop() error {
	c.Logger.Info(fmt.Sprintf("Stopping the http exporter %s", c.Config.Name))
	return nil
}

// Name returns the name of the exporter
func (c *HTTPExporter) Name() string {
	return c.Config.Name
}

// GetConfig returns the config of the exporter
func (c *HTTPExporter) GetConfig() interface{} {
	return c.Config
}

// Push pushes events to the HTTP destination
func (c *HTTPExporter) Push(result *healthcheck.Result) error {
	var jsonBytes []byte
	payload := []*healthcheck.Result{result}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "Fail to convert result to json:\n%v", result)
	}
	req, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return errors.Wrapf(err, "HTTP exporter: fail to send healthchecks to %s", c.URL)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "HTTP exporter: fail to send healthchecks to %s", c.URL)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP exporter: request failed, status %d", resp.StatusCode)
	}
	return nil
}
