package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/discovery"
	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/memorystore"
	"github.com/mcorbin/cabourotte/prometheus"
)

// Component the http server component
type Component struct {
	MemoryStore      *memorystore.MemoryStore
	Config           *Configuration
	DiscoveryConfig  *discovery.Configuration
	Logger           *zap.Logger
	healthcheck      *healthcheck.Component
	Server           *echo.Echo
	Prometheus       *prometheus.Prometheus
	requestHistogram *prom.HistogramVec
	responseCounter  *prom.CounterVec
}

// New creates a new HTTP component
func New(logger *zap.Logger, memstore *memorystore.MemoryStore, promComponent *prometheus.Prometheus, config *Configuration, discoveryConfig *discovery.Configuration, healthcheck *healthcheck.Component) (*Component, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	if config.Cert != "" {
		caCert, err := ioutil.ReadFile(config.Cacert)
		if err != nil {
			return nil, errors.Wrap(err, "fail to read the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Create the TLS Config with the CA pool and enable Client certificate validation
		tlsConfig := &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}

		serverCert, err := ioutil.ReadFile(config.Cert)
		if err != nil {
			return nil, errors.Wrap(err, "fail to read the certificate cert")
		}

		serverKey, err := ioutil.ReadFile(config.Key)
		if err != nil {
			return nil, errors.Wrap(err, "fail to read the certificate key")
		}

		x509KkeyPair, err := tls.X509KeyPair(serverCert, serverKey)
		if err != nil {
			return nil, errors.Wrap(err, "fail to build the x509 keypair")
		}

		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0] = x509KkeyPair
		s := e.TLSServer
		s.TLSConfig = tlsConfig
	}

	respCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "http_responses_total",
			Help: "Count the number of HTTP responses.",
		},
		[]string{"method", "status", "path"})

	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}

	reqHistogram := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "http_requests_duration_second",
			Help:    "Time to execute http requests",
			Buckets: buckets,
		},
		[]string{"method", "path"})

	component := Component{
		MemoryStore:      memstore,
		DiscoveryConfig:  discoveryConfig,
		Config:           config,
		Server:           e,
		Logger:           logger,
		healthcheck:      healthcheck,
		Prometheus:       promComponent,
		requestHistogram: reqHistogram,
		responseCounter:  respCounter,
	}
	return &component, nil
}

// Start starts the http server
func (c *Component) Start() error {
	address := fmt.Sprintf("%s:%d", c.Config.Host, c.Config.Port)
	c.Logger.Info(fmt.Sprintf("Starting the HTTP server component on %s", address))
	c.handlers()
	err := c.Prometheus.Register(c.responseCounter)
	if err != nil {
		return errors.Wrapf(err, "fail to register the Prometheus HTTP response counter")
	}
	err = c.Prometheus.Register(c.requestHistogram)
	if err != nil {
		return errors.Wrapf(err, "fail to register the Prometheus HTTP request histogram")
	}
	go func() {
		var err error
		if c.Config.Cert != "" {
			c.Logger.Info("TLS enabled")
			s := c.Server.TLSServer
			s.Addr = address
			if !c.Server.DisableHTTP2 {
				s.TLSConfig.NextProtos = append(s.TLSConfig.NextProtos, "h2")
			}
			err = c.Server.StartServer(s)
		} else {
			err = c.Server.Start(address)
		}
		if err != http.ErrServerClosed {
			c.Logger.Error(fmt.Sprintf("HTTP server error: %s", err.Error()))
			os.Exit(2)
		}
	}()
	// todo: remove this, causes issues in tests
	time.Sleep(300 * time.Millisecond)
	return nil
}

// Stop stop the server compoment
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the HTTP server component")
	c.Prometheus.Unregister(c.requestHistogram)
	c.Prometheus.Unregister(c.responseCounter)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := c.Server.Shutdown(ctx)
	if err != nil {
		c.Logger.Error(err.Error())
		return err
	}
	c.Logger.Info("HTTP server stopped")
	return nil
}
