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
		if err != nil {
			context.Error(err)
		}
		status := fmt.Sprintf("%d", context.Response().Status)
		if status == "404" {
			path = "?"
		}
		c.requestHistogram.With(prom.Labels{"method": method, "path": path}).Observe(duration.Seconds())
		c.responseCounter.With(prom.Labels{"method": method, "status": status, "path": path}).Inc()
		return nil
	}
}
