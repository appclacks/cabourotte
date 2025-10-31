package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/appclacks/cabourotte/healthcheck"
	"github.com/appclacks/cabourotte/tls"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"
)

// HTTPDiscovery the http discovery struct
type HTTPDiscovery struct {
	Logger           *zap.Logger
	requestHistogram *prom.HistogramVec
	responseCounter  *prom.CounterVec
	Healthcheck      *healthcheck.Component
	URL              string
	Config           *Configuration
	Client           *http.Client
	t                tomb.Tomb
	tick             *time.Ticker
}

// New creates a new HTTP Discovery
func New(logger *zap.Logger, config *Configuration, checkComponent *healthcheck.Component, counter *prom.CounterVec, histogram *prom.HistogramVec) (*HTTPDiscovery, error) {
	protocol := "http"
	tlsConfig, err := tls.GetTLSConfig(config.Key, config.Cert, config.Cacert, "", config.Insecure)
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

	component := HTTPDiscovery{
		Healthcheck:      checkComponent,
		responseCounter:  counter,
		requestHistogram: histogram,
		Logger:           logger,
		Config:           config,
		URL:              url,
		Client: &http.Client{
			Transport: otelhttp.NewTransport(
				transport,
				otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
					return otelhttptrace.NewClientTrace(ctx)
				}),
			),
			Timeout: time.Second * 5,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return &component, nil
}

func (c *HTTPDiscovery) request(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "HTTP discovery: fail to create request for %s", c.URL)
	}
	req.Header.Set("User-Agent", "Cabourotte")
	for k, v := range c.Config.Headers {
		req.Header.Set(k, v)
	}
	if len(c.Config.Query) != 0 {
		q := req.URL.Query()
		for k, v := range c.Config.Query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "HTTP discovery: fail to send request to %s", c.URL)
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP discovery: request failed, status %d", resp.StatusCode)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Fail to read request body")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP Discovery: request failed, status %d, body %s", resp.StatusCode, string(responseBody))
	}
	var payload ResultPayload
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return fmt.Errorf("HTTP Discovery: fail to convert the payload from json: %s", err.Error())
	}
	return c.Healthcheck.ReloadForSource(
		fmt.Sprintf("%s-%s", healthcheck.SourceHTTPDiscovery, c.Config.Name),
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
				tracer := otel.Tracer("discovery")
				ctx, span := tracer.Start(context.Background(), "discovery")
				span.SetAttributes(attribute.String("cabourotte.discovery.name", c.Config.Name))
				span.SetAttributes(attribute.String("cabourotte.discovery.type", "http"))
				c.Logger.Debug(fmt.Sprintf("HTTP discovery: polling %s", c.URL))
				start := time.Now()
				status := "success"
				err := c.request(ctx)
				duration := time.Since(start)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, "discovery failure")
					status = "failure"
					msg := fmt.Sprintf("HTTP discovery error: %s", err.Error())
					c.Logger.Error(msg)
				} else {
					span.SetStatus(codes.Ok, "discovery successful")
				}
				span.SetAttributes(attribute.String("cabourotte.discovery.status", status))
				span.End()
				c.requestHistogram.With(prom.Labels{"name": c.Config.Name}).Observe(duration.Seconds())
				c.responseCounter.With(prom.Labels{"status": status, "name": c.Config.Name}).Inc()
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
