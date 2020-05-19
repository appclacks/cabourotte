package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
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
func NewHTTPExporter(logger *zap.Logger, config *HTTPConfiguration) *HTTPExporter {
	protocol := "http"
	if config.Protocol == healthcheck.HTTPS {
		protocol = "https"
	}
	url := fmt.Sprintf(
		"%s://%s%s",
		protocol,
		net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
		config.Path)
	exporter := HTTPExporter{
		Logger: logger,
		Config: config,
		URL:    url,
		Client: &http.Client{
			Timeout: time.Second * 3,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return &exporter
}

// Start starts the HTTP exporter component
func (c *HTTPExporter) Start() error {
	// nothing to do
	c.Logger.Info(fmt.Sprintf("Starting the HTTP healcheck exporter on %s:%d", c.Config.Host, c.Config.Port))
	return nil
}

// Stop stops the HTTP exporter component
func (c *HTTPExporter) Stop() error {
	return nil
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
