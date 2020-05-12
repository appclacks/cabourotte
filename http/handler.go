package http

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"

	"cabourotte/healthcheck"
)

func (c *Component) addCheck(ec echo.Context, healthcheck healthcheck.Healthcheck) error {
	err := c.healthcheck.AddCheck(healthcheck)
	if err != nil {
		msg := fmt.Sprintf("Fail to start the healthcheck: %s", err.Error())
		c.Logger.Error(msg)
		return ec.JSON(http.StatusInternalServerError, msg)
	}
	return nil
}

// handlers configures the handlers for the http server component
// todo: handler one-off tasks
func (c *Component) handlers() {
	c.Server.POST("/healthcheck/dns", func(ec echo.Context) error {
		var config healthcheck.DNSHealthcheckConfiguration
		if err := ec.Bind(&config); err != nil {
			msg := fmt.Sprintf("Fail to create the dns healthcheck. Invalid JSON: %s", err.Error())
			c.Logger.Error(msg)
			return ec.JSON(http.StatusBadRequest, msg)
		}
		healthcheck := healthcheck.NewDNSHealthcheck(c.Logger, &config)
		return c.addCheck(ec, healthcheck)

	})
	c.Server.POST("/healthcheck/tcp", func(ec echo.Context) error {
		var config healthcheck.TCPHealthcheckConfiguration
		if err := ec.Bind(&config); err != nil {
			msg := fmt.Sprintf("Fail to create the TCP healthcheck. Invalid JSON: %s", err.Error())
			c.Logger.Error(msg)
			return ec.JSON(http.StatusBadRequest, msg)
		}
		healthcheck := healthcheck.NewTCPHealthcheck(c.Logger, &config)
		return c.addCheck(ec, healthcheck)
		return nil
	})
	c.Server.POST("/healthcheck/http", func(ec echo.Context) error {
		var config healthcheck.HTTPHealthcheckConfiguration
		if err := ec.Bind(&config); err != nil {
			msg := fmt.Sprintf("Fail to create the HTTP healthcheck. Invalid JSON: %s", err.Error())
			c.Logger.Error(msg)
			return ec.JSON(http.StatusBadRequest, msg)
		}
		healthcheck := healthcheck.NewHTTPHealthcheck(c.Logger, &config)
		return c.addCheck(ec, healthcheck)
		return nil
	})
	c.Server.GET("/healthcheck", func(ec echo.Context) error {
		return nil
	})
}
