package http

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/mcorbin/gadgeto/tonic"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	ierror "github.com/mcorbin/httpgo/error"
)

var HTTPCodes = map[ierror.ErrorType]int{
	ierror.BadRequest:   400,
	ierror.Unauthorized: 401,
	ierror.Forbidden:    403,
	ierror.NotFound:     404,
	ierror.Conflict:     409,
	ierror.Internal:     500,
}

type ErrorResponse struct {
	Messages []string `json:"messages"`
}

func DefaultBindingHookMaxBodyBytes(maxBodyBytes int64) tonic.BindHook {
	return func(c *gin.Context, i interface{}) error {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		if c.Request.ContentLength == 0 || c.Request.Method == http.MethodGet {
			return nil
		}
		if err := c.ShouldBindWith(i, binding.JSON); err != nil && err != io.EOF {

			jsonError, ok := err.(*json.UnmarshalTypeError)
			if ok {
				return fmt.Errorf("Invalid value for field %s", jsonError.Field)
			}
			return ierror.New("Invalid JSON", ierror.BadRequest, true)
		}
		return nil
	}
}

func ErrorHook(logger *zap.Logger) func(c *gin.Context, err error) (int, interface{}) {
	defaultMsg := "Internal error"
	invalidParameterMsg := "Invalid parameters"
	return func(c *gin.Context, err error) (int, interface{}) {
		response := ErrorResponse{}
		status := 500
		internalError, ok := err.(ierror.Error)
		if ok {
			if s, ok := HTTPCodes[internalError.Type]; ok {
				status = s
			}
			if len(internalError.Messages) != 0 && internalError.Exposable {
				response.Messages = internalError.Messages
			} else {
				response.Messages = []string{defaultMsg}
			}
		}
		if strings.Contains(err.Error(), "Invalid value for field") {
			response.Messages = []string{err.Error()}
			status = 400

		} else if strings.Contains(err.Error(), "binding error") {
			status = 400
			bindError, ok := err.(tonic.BindError)
			if ok {
				validationErrors := bindError.ValidationErrors()
				if len(validationErrors) == 0 {
					response.Messages = []string{invalidParameterMsg}
				}
				for _, e := range validationErrors {
					msg := fmt.Sprintf("Invalid field %s (path %s)", e.Field(), e.Namespace())
					response.Messages = append(response.Messages, msg)
				}
			} else {
				response.Messages = []string{invalidParameterMsg}
			}
		}
		if len(response.Messages) == 0 {
			response.Messages = []string{defaultMsg}
		}
		logger.Error(err.Error())
		return status, response
	}
}
