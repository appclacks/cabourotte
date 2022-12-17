package http

import (
	"fmt"
	"time"

	"github.com/labstack/echo"
	prom "github.com/prometheus/client_golang/prometheus"
)

// countReq count the bumber of requests to the server
func (c *Component) metricMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		start := time.Now()
		err := next(context)
		if err != nil {
			// all configured error handler
			// to populate the response
			context.Error(err)
		}
		duration := time.Since(start)
		method := context.Request().Method
		path := context.Path()
		response := context.Response()
		if response != nil {
			status := fmt.Sprintf("%d", context.Response().Status)
			if status == "404" {
				path = "?"
			}
			c.requestHistogram.With(prom.Labels{"method": method, "path": path}).Observe(duration.Seconds())
			c.responseCounter.With(prom.Labels{"method": method, "status": status, "path": path}).Inc()
		} else {
			c.Logger.Error(fmt.Sprintf("Response in metrics middleware is nil for %s %s", method, path))
		}
		// return nil car we already called the error handler middleware here
		return nil
	}
}
