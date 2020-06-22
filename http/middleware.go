package http

import (
	"fmt"
	"time"

	"github.com/labstack/echo"
	prom "github.com/prometheus/client_golang/prometheus"
)

// countReq count the bumber of requests to the server
func (c *Component) countResponse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		start := time.Now()
		err := next(context)
		duration := time.Since(start)
		method := context.Request().Method
		path := context.Path()
		status := fmt.Sprintf("%d", context.Response().Status)
		if err != nil {
			c.Logger.Error(status)
			context.Error(err)
		}
		if status == "404" {
			path = "?"
		}
		c.requestHistogram.With(prom.Labels{"method": method, "path": path}).Observe(duration.Seconds())
		c.requestCounter.With(prom.Labels{"method": method, "path": path}).Inc()
		c.responseCounter.With(prom.Labels{"method": method, "status": status, "path": path}).Inc()
		return nil
	}
}
