package http

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"net/http"
	"text/template"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"cabourotte/healthcheck"
)

// BasicResponse a type for HTTP responses
type BasicResponse struct {
	Message string `json:"message"`
}

// addCheck adds a periodic healthcheck to the healthcheck component.
func (c *Component) addCheck(ec echo.Context, check healthcheck.Healthcheck) error {
	check.SetSource(healthcheck.API)
	err := c.healthcheck.AddCheck(check)
	if err != nil {
		return err
	}
	return nil
}

// oneOff executes an one-off healthcheck and returns its result
func (c *Component) oneOff(ec echo.Context, healthcheck healthcheck.Healthcheck) error {
	c.Logger.Info(fmt.Sprintf("Executing one-off healthcheck %s", healthcheck.Base().Name))
	err := healthcheck.Initialize()
	if err != nil {
		msg := fmt.Sprintf("Fail to initialize one off healthcheck %s: %s", healthcheck.Base().Name, err.Error())
		c.Logger.Error(msg)
		return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: msg})
	}
	err = healthcheck.Execute()
	if err != nil {
		msg := fmt.Sprintf("Execution of one off healthcheck %s failed: %s", healthcheck.Base().Name, err.Error())
		c.Logger.Error(msg)
		return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: msg})
	}
	msg := fmt.Sprintf("One-off healthcheck %s successfully executed", healthcheck.Base().Name)
	c.Logger.Info(msg)
	return ec.JSON(http.StatusCreated, &BasicResponse{Message: msg})
}

func (c *Component) addCheckError(ec echo.Context, healthcheck healthcheck.Healthcheck, err error) error {
	msg := fmt.Sprintf("Fail to start the healthcheck %s: %s", healthcheck.Base().Name, err.Error())
	c.Logger.Error(msg)
	return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: msg})
}

// handleCheck handles new healthchecks requests
func (c *Component) handleCheck(ec echo.Context, healthcheck healthcheck.Healthcheck) error {
	if healthcheck.Base().OneOff {
		return c.oneOff(ec, healthcheck)
	}
	err := c.addCheck(ec, healthcheck)
	if err != nil {
		return c.addCheckError(ec, healthcheck, err)
	}
	return ec.JSON(http.StatusCreated, &BasicResponse{Message: "Healthcheck successfully added"})
}

// handlers configures the handlers for the http server component
func (c *Component) handlers() {
	c.Server.Use(c.countResponse)
	if c.Config.BasicAuth.Username != "" {
		c.Server.Use(middleware.BasicAuth(func(username, password string, ctx echo.Context) (bool, error) {
			if subtle.ConstantTimeCompare([]byte(username),
				[]byte(c.Config.BasicAuth.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(password),
					[]byte(c.Config.BasicAuth.Password)) == 1 {
				return true, nil
			}
			c.Logger.Error("Invalid Basic Auth credentials")
			return false, nil
		}))
	}
	echo.NotFoundHandler = func(ec echo.Context) error {
		return ec.JSON(http.StatusNotFound, &BasicResponse{Message: "not found"})
	}
	if !c.Config.DisableHealthcheckAPI {
		c.Server.POST("/healthcheck/dns", func(ec echo.Context) error {
			var config healthcheck.DNSHealthcheckConfiguration
			if err := ec.Bind(&config); err != nil {
				msg := fmt.Sprintf("Fail to create the dns healthcheck. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := config.Validate()
			if err != nil {
				msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			healthcheck := healthcheck.NewDNSHealthcheck(c.Logger, &config)
			return c.handleCheck(ec, healthcheck)
		})

		c.Server.POST("/healthcheck/tcp", func(ec echo.Context) error {
			var config healthcheck.TCPHealthcheckConfiguration
			if err := ec.Bind(&config); err != nil {
				msg := fmt.Sprintf("Fail to create the TCP healthcheck. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := config.Validate()
			if err != nil {
				msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			healthcheck := healthcheck.NewTCPHealthcheck(c.Logger, &config)
			return c.handleCheck(ec, healthcheck)
		})

		c.Server.POST("/healthcheck/tls", func(ec echo.Context) error {
			var config healthcheck.TLSHealthcheckConfiguration
			if err := ec.Bind(&config); err != nil {
				msg := fmt.Sprintf("Fail to create the TLS healthcheck. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := config.Validate()
			if err != nil {
				msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			healthcheck := healthcheck.NewTLSHealthcheck(c.Logger, &config)
			return c.handleCheck(ec, healthcheck)
		})

		c.Server.POST("/healthcheck/http", func(ec echo.Context) error {
			var config healthcheck.HTTPHealthcheckConfiguration
			if err := ec.Bind(&config); err != nil {
				msg := fmt.Sprintf("Fail to create the HTTP healthcheck. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := config.Validate()
			if err != nil {
				msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			healthcheck := healthcheck.NewHTTPHealthcheck(c.Logger, &config)
			return c.handleCheck(ec, healthcheck)
		})

		c.Server.POST("/healthcheck/command", func(ec echo.Context) error {
			var config healthcheck.CommandHealthcheckConfiguration
			if err := ec.Bind(&config); err != nil {
				msg := fmt.Sprintf("Fail to create the Command healthcheck. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := config.Validate()
			if err != nil {
				msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			healthcheck := healthcheck.NewCommandHealthcheck(c.Logger, &config)
			return c.handleCheck(ec, healthcheck)
		})

		c.Server.POST("/healthcheck/bulk", func(ec echo.Context) error {
			var payload BulkPayload
			if err := ec.Bind(&payload); err != nil {
				msg := fmt.Sprintf("Fail to add healthchecks. Invalid JSON: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			err := payload.Validate()
			if err != nil {
				msg := fmt.Sprintf("Fail to validate healthchecks configuration: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusBadRequest, &BasicResponse{Message: msg})
			}
			for _, config := range payload.HTTPChecks {
				healthcheck := healthcheck.NewHTTPHealthcheck(c.Logger, &config)
				err := c.addCheck(ec, healthcheck)
				if err != nil {
					return c.addCheckError(ec, healthcheck, err)
				}
			}
			for _, config := range payload.TCPChecks {
				healthcheck := healthcheck.NewTCPHealthcheck(c.Logger, &config)
				err := c.addCheck(ec, healthcheck)
				if err != nil {
					return c.addCheckError(ec, healthcheck, err)
				}
			}
			for _, config := range payload.DNSChecks {
				healthcheck := healthcheck.NewDNSHealthcheck(c.Logger, &config)
				err := c.addCheck(ec, healthcheck)
				if err != nil {
					return c.addCheckError(ec, healthcheck, err)
				}
			}
			for _, config := range payload.TLSChecks {
				healthcheck := healthcheck.NewTLSHealthcheck(c.Logger, &config)
				err := c.addCheck(ec, healthcheck)
				if err != nil {
					return c.addCheckError(ec, healthcheck, err)
				}
			}
			for _, config := range payload.CommandChecks {
				healthcheck := healthcheck.NewCommandHealthcheck(c.Logger, &config)
				err := c.addCheck(ec, healthcheck)
				if err != nil {
					return c.addCheckError(ec, healthcheck, err)
				}
			}
			return ec.JSON(http.StatusCreated, &BasicResponse{Message: "Healthchecks successfully added"})
		})

		c.Server.GET("/healthcheck", func(ec echo.Context) error {
			return ec.JSON(http.StatusOK, c.healthcheck.ListChecks())
		})
		c.Server.GET("/healthcheck/:name", func(ec echo.Context) error {
			name := ec.Param("name")
			healthcheck, err := c.healthcheck.GetCheck(name)
			if err != nil {
				return ec.JSON(http.StatusNotFound, &BasicResponse{Message: err.Error()})
			}
			return ec.JSON(http.StatusOK, healthcheck)
		})

		c.Server.DELETE("/healthcheck/:name", func(ec echo.Context) error {
			name := ec.Param("name")
			c.Logger.Info(fmt.Sprintf("Deleting healthcheck %s", name))
			err := c.healthcheck.RemoveCheck(name)
			if err != nil {
				msg := fmt.Sprintf("Fail to start the healthcheck: %s", err.Error())
				c.Logger.Error(msg)
				return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: msg})
			}
			return ec.JSON(http.StatusOK, &BasicResponse{Message: fmt.Sprintf("Successfully deleted healthcheck %s", name)})
		})
	}
	if !c.Config.DisableResultAPI {
		c.Server.GET("/result", func(ec echo.Context) error {
			return ec.JSON(http.StatusOK, c.MemoryStore.List())
		})
		c.Server.GET("/result/:name", func(ec echo.Context) error {
			name := ec.Param("name")
			result, err := c.MemoryStore.Get(name)
			if err != nil {
				return ec.JSON(http.StatusNotFound, &BasicResponse{Message: err.Error()})
			}
			return ec.JSON(http.StatusOK, result)

		})

		c.Server.GET("/frontend", func(ec echo.Context) error {
			tmpl, err := template.New("frontend").Parse(frontendTemplate)
			if err != nil {
				c.Logger.Error(err.Error())
				return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: err.Error()})
			}
			var tmplBytes bytes.Buffer
			if err := tmpl.Execute(&tmplBytes, c.MemoryStore.List()); err != nil {
				c.Logger.Error(err.Error())
				return ec.JSON(http.StatusInternalServerError, &BasicResponse{Message: err.Error()})
			}
			return ec.HTML(http.StatusOK, tmplBytes.String())
		})
	}

	c.Server.GET("/health", func(ec echo.Context) error {
		return ec.JSON(http.StatusOK, "ok")
	})

	c.Server.GET("/healthz", func(ec echo.Context) error {
		return ec.JSON(http.StatusOK, "ok")
	})

	c.Server.GET("/metrics", echo.WrapHandler(c.Prometheus.Handler()))
}
