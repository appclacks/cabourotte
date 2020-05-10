package healthcheck

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// Protocol is the healthcheck http protocol
type Protocol int

const (
	// HTTP the HTTP protocol
	HTTP Protocol = 1 + iota
	// HTTPS the HTTPS protocol
	HTTPS
)

// HTTPHealthcheckConfiguration defines an HTTP healthcheck configuration
type HTTPHealthcheckConfiguration struct {
	Name        string
	ValidStatus []uint
	Description string
	// can be an IP or a domain
	Target   string
	Port     uint
	Protocol Protocol
	Path     string
	Timeout  time.Duration
	Interval time.Duration
	OneOff   bool
}

// HTTPHealthcheck defines an HTTP healthcheck
type HTTPHealthcheck struct {
	Logger *zap.Logger
	ID     string
	config *HTTPHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// buildURL build the target URL for the HTTP healthcheck, depending of its
// configuration
func (h *HTTPHealthcheck) buildURL() {
	protocol := "http"
	if h.config.Protocol == HTTPS {
		protocol = "https"
	}
	h.URL = fmt.Sprintf("%s://%s:%d%s", protocol, h.config.Target, h.config.Port, h.config.Path)
}

// Start an Healthcheck, which will be periodically executed after a
//  given interval of time
func (h *HTTPHealthcheck) Start() error {
	h.Tick = time.NewTicker(time.Duration(h.config.Interval))
	h.buildURL()
	h.t.Go(func() error {
		for {
			select {
			case <-h.Tick.C:
				h.Execute()
			case <-h.t.Dying():
				return nil
			}
		}
	})
	return nil
}

// Stop an Healthcheck
func (h *HTTPHealthcheck) Stop() error {
	h.Tick.Stop()
	h.t.Kill(nil)
	h.t.Wait()
	return nil

}

// isSuccessful verifies if a healthcheck result is considered valid
// depending of the healthcheck configuration
func (h *HTTPHealthcheck) isSuccessful(response *http.Response) bool {
	for _, s := range h.config.ValidStatus {
		if uint(response.StatusCode) == s {
			return true
		}
	}
	return false
}

// logError logs an error with context
func (h *HTTPHealthcheck) logError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("url", h.URL),
		zap.String("name", h.config.Name),
		zap.String("id", h.ID))
}

// Execute executes an healthcheck on the given target
func (h *HTTPHealthcheck) Execute() error {
	ctx := h.t.Context(nil)
	req, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "fail to initialize HTTP request")
	}
	req.Header.Set("User-Agent", "Cabourotte")
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()
	req = req.WithContext(timeoutCtx)
	response, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "HTTP request failed")
	}
	defer response.Body.Close()
	if !h.isSuccessful(response) {
		body, readErr := ioutil.ReadAll(response.Body)
		if readErr != nil {
			return errors.Wrapf(readErr, "Fail to read request body")
		}
		bodyStr := string(body)
		errorMsg := fmt.Sprintf("HTTP request failed: %d %s", response.StatusCode, bodyStr)
		err = errors.New(errorMsg)
		return err
	}
	return nil
}
