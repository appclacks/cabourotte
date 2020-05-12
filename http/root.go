package http

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo"
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
	s := echo.New()
	s.HideBanner = true
	component := Component{
		Config:      config,
		Server:      s,
		Logger:      logger,
		healthcheck: healthcheck,
	}
	return &component, nil
}

// Start starts the http server
func (c *Component) Start() error {
	c.Logger.Info("Starting the HTTP server component")
	address := fmt.Sprintf("%s:%d", c.Config.Host, c.Config.Port)
	go func() {
		err := c.Server.Start(address)
		if err != nil {
			c.Logger.Info("Stopping the HTTP server")
		}
	}()
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
