package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
	"github.com/mcorbin/cabourotte/tls"
)

// HTTPDiscovery the http discovery struct
type HTTPDiscovery struct {
	Logger           *zap.Logger
	requestHistogram *prom.HistogramVec
	Healthcheck      *healthcheck.Component
	URL              string
	Config           *Configuration
	Client           *http.Client
	t                tomb.Tomb
	tick             *time.Ticker
}

// New creates a new HTTP Discovery
func New(logger *zap.Logger, config *Configuration, checkComponent *healthcheck.Component, promComponent *prometheus.Prometheus) (*HTTPDiscovery, error) {
	protocol := "http"
	tlsConfig, err := tls.GetTLSConfig(config.Key, config.Cert, config.Cacert, config.Insecure)
	if err != nil {
		return nil, err
	}
	if config.Protocol == healthcheck.HTTPS {
		protocol = "https"
	}
	url := fmt.Sprintf(
		"%s://%s%s",
		protocol,
		net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
		config.Path)
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}
	histo := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    "http_discovery_duration_seconds",
		Help:    "Time to execute the HTTP request for healthchecks discovery.",
		Buckets: buckets,
	},
		[]string{"status"},
	)
	err = promComponent.Register(histo)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the http discovery request histogram")
	}
	component := HTTPDiscovery{
		Healthcheck:      checkComponent,
		requestHistogram: histo,
		Logger:           logger,
		Config:           config,
		URL:              url,
		Client: &http.Client{
			Transport: transport,
			Timeout:   time.Second * 5,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return &component, nil
}

func (c *HTTPDiscovery) request() error {
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "HTTP discovery: fail to create request for %s", c.URL)
	}
	req.Header.Set("User-Agent", "Cabourotte")
	for k, v := range c.Config.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "HTTP discovery: fail to send request to %s", c.URL)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP discovery: request failed, status %d", resp.StatusCode)
	}
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Fail to read request body")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP Discovery: request failed, status %d, body %s", resp.StatusCode, string(responseBody))
	}
	var payload ResultPayload
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return fmt.Errorf("HTTP Discovery: fail to convert the payload %s from json", string(responseBody))
	}
	return c.Healthcheck.ReloadForSource(
		healthcheck.SourceHTTPDiscovery,
		nil,
		payload.CommandChecks,
		payload.DNSChecks,
		payload.TCPChecks,
		payload.HTTPChecks,
		payload.TLSChecks)
}

// Start starts the HTTP discovery component
func (c *HTTPDiscovery) Start() error {
	c.tick = time.NewTicker(time.Duration(c.Config.Interval))
	c.t.Go(func() error {
		c.Logger.Info(fmt.Sprintf("Starting the HTTP healthcheck discovery on %s:%d", c.Config.Host, c.Config.Port))
		for {
			select {
			case <-c.tick.C:
				c.Logger.Debug(fmt.Sprintf("HTTP discovery: polling %s", c.URL))
				start := time.Now()
				status := "success"
				err := c.request()
				duration := time.Since(start)
				if err != nil {
					status = "failure"
					msg := fmt.Sprintf("HTTP discovery error: %s", err.Error())
					c.Logger.Error(msg)
				}
				c.requestHistogram.With(prom.Labels{"status": status}).Observe(duration.Seconds())
			case <-c.t.Dying():
				return nil
			}
		}
	})
	return nil
}

// Stop stops the HTTP discovery component
func (c *HTTPDiscovery) Stop() error {
	c.Logger.Info("Stopping the http discovery")
	c.tick.Stop()
	c.t.Kill(nil)
	err := c.t.Wait()
	if err != nil {
		return err
	}
	return nil
}
