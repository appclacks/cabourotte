package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

// Component the http server component
type Component struct {
	Config      *Configuration
	Logger      *zap.Logger
	healthcheck *healthcheck.Component
	Server      *echo.Echo
}

// New creates a new HTTP component
func New(logger *zap.Logger, config *Configuration, healthcheck *healthcheck.Component) (*Component, error) {
	e := echo.New()
	e.HideBanner = true
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
		tlsConfig.BuildNameToCertificate()
		s := e.TLSServer
		s.TLSConfig = tlsConfig
	}

	component := Component{
		Config:      config,
		Server:      e,
		Logger:      logger,
		healthcheck: healthcheck,
	}
	return &component, nil
}

// Start starts the http server
func (c *Component) Start() error {
	address := fmt.Sprintf("%s:%d", c.Config.Host, c.Config.Port)
	c.Logger.Info(fmt.Sprintf("Starting the HTTP server component on %s", address))
	c.handlers()
	go func() {

		if c.Config.Cert != "" {

			c.Server.StartTLS(address, c.Config.Cert, c.Config.Key)
		} else {
			c.Server.Start(address)
		}
	}()
	// todo: remove this, causes issues in tests
	time.Sleep(300 * time.Millisecond)
	return nil
}

// Stop stop the server compoment
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the HTTP server component")
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
