package http

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/labstack/echo"
	"github.com/mcorbin/corbierror"
)

func errorHandler(logger *zap.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		logger.Error(fmt.Sprintf("HTTP error: %s", err.Error()))
		if he, ok := err.(*echo.HTTPError); ok {
			if he.Code == 401 {
				err := c.JSON(401, corbierror.New("Unauthorized", corbierror.Unauthorized, true))
				if err != nil {
					logger.Error(err.Error())
				}
				return
			}
		}
		if e, ok := err.(*corbierror.Error); ok {
			httpErr, status := corbierror.HTTPError(*e)
			err := c.JSON(status, httpErr)
			if err != nil {
				logger.Error(err.Error())
			}
			return
		}
		err = c.JSON(500, corbierror.New("Internal error", corbierror.Internal, true))
		if err != nil {
			logger.Error(err.Error())
		}
	}
}
