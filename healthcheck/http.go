package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// HTTPHealthcheckConfiguration defines an HTTP healthcheck configuration
type HTTPHealthcheckConfiguration struct {
	Name        string `json:"name"`
	ValidStatus []uint `json:"valid-status" yaml:"valid_status"`
	Description string `json:"description"`
	// can be an IP or a domain
	Target   string   `json:"target"`
	Port     uint     `json:"port"`
	Protocol Protocol `json:"protocol"`
	Path     string   `json:"path"`
	Timeout  Duration `json:"timeout"`
	Interval Duration `json:"interval"`
	OneOff   bool     `json:"one-off,"`
	Key      string   `json:"key,omitempty"`
	Cert     string   `json:"cert,omitempty"`
	Cacert   string   `json:"cacert,omitempty"`
}

// ValidateHTTPConfig validates the healthcheck configuration
func ValidateHTTPConfig(config *HTTPHealthcheckConfiguration) error {
	if config.Name == "" {
		return errors.New("The healthcheck name is missing")
	}
	if len(config.ValidStatus) == 0 {
		return errors.New("At least one valid status code should be provided")
	}
	if config.Target == "" {
		return errors.New("The healthcheck target is missing")
	}
	if config.Port == 0 {
		return errors.New("The healthcheck port is missing")
	}
	if config.Timeout == 0 {
		return errors.New("The healthcheck timeout is missing")
	}
	if config.Interval < Duration(2*time.Second) {
		return errors.New("The healthcheck interval should be greater than 2 second")
	}
	if config.Interval < config.Timeout {
		return errors.New("The healthcheck interval should be greater than the timeout")
	}
	if !((config.Key != "" && config.Cert != "" && config.Cacert != "") ||
		(config.Key == "" && config.Cert == "" && config.Cacert == "")) {
		return errors.New("Invalid certificates")
	}
	return nil
}

// HTTPHealthcheck defines an HTTP healthcheck
type HTTPHealthcheck struct {
	Logger *zap.Logger
	Config *HTTPHealthcheckConfiguration
	URL    string

	Tick      *time.Ticker
	t         tomb.Tomb
	transport *http.Transport
}

// buildURL build the target URL for the HTTP healthcheck, depending of its
// configuration
func (h *HTTPHealthcheck) buildURL() {
	protocol := "http"
	if h.Config.Protocol == HTTPS {
		protocol = "https"
	}
	h.URL = fmt.Sprintf(
		"%s://%s%s",
		protocol,
		net.JoinHostPort(h.Config.Target, fmt.Sprintf("%d", h.Config.Port)),
		h.Config.Path)
}

// Name returns the healthcheck identifier.
func (h *HTTPHealthcheck) Name() string {
	return h.Config.Name
}

// Initialize the healthcheck.
func (h *HTTPHealthcheck) Initialize() error {
	h.buildURL()
	// tls is enabled
	if h.Config.Key != "" {
		cert, err := tls.LoadX509KeyPair(h.Config.Cert, h.Config.Key)
		if err != nil {
			return errors.Wrapf(err, "Fail to load certificates")
		}
		caCert, err := ioutil.ReadFile(h.Config.Cacert)
		if err != nil {
			return errors.Wrapf(err, "Fail to load the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		h.transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		}
	} else {
		h.transport = &http.Transport{}
	}
	return nil
}

// OneOff returns true if the healthcheck if a one-off check
func (h *HTTPHealthcheck) OneOff() bool {
	return h.Config.OneOff

}

// Interval Get the interval.
func (h *HTTPHealthcheck) Interval() Duration {
	return h.Config.Interval
}

// isSuccessful verifies if a healthcheck result is considered valid
// depending of the healthcheck configuration
func (h *HTTPHealthcheck) isSuccessful(response *http.Response) bool {
	for _, s := range h.Config.ValidStatus {
		if uint(response.StatusCode) == s {
			return true
		}
	}
	return false
}

// LogError logs an error with context
func (h *HTTPHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogDebug logs a message with context
func (h *HTTPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogInfo logs a message with context
func (h *HTTPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// Execute executes an healthcheck on the given target
func (h *HTTPHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx := h.t.Context(nil)
	req, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "fail to initialize HTTP request")
	}
	req.Header.Set("User-Agent", "Cabourotte")
	client := &http.Client{
		Transport: h.transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.Timeout))
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

// NewHTTPHealthcheck creates a HTTP healthcheck from a logger and a configuration
func NewHTTPHealthcheck(logger *zap.Logger, config *HTTPHealthcheckConfiguration) *HTTPHealthcheck {
	return &HTTPHealthcheck{
		Logger: logger,
		Config: config,
	}
}

// MarshalJSON marshal to json a dns healthcheck
func (h *HTTPHealthcheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}
