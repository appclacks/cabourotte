package healthcheck

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// HTTPHealthcheckConfiguration defines an HTTP healthcheck configuration
type HTTPHealthcheckConfiguration struct {
	Name        string `json:"name"`
	ValidStatus []uint `json:"valid-status" yaml:"valid-status"`
	Description string `json:"description"`
	// can be an IP or a domain
	Target     string            `json:"target"`
	Port       uint              `json:"port"`
	Redirect   bool              `json:"redirect"`
	Body       string            `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Protocol   Protocol          `json:"protocol"`
	Path       string            `json:"path,omitempty"`
	SourceIP   IP                `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	BodyRegexp []Regexp          `json:"body-regexp,omitempty" yaml:"body-regexp,omitempty"`
	Insecure   bool
	Timeout    Duration          `json:"timeout"`
	Interval   Duration          `json:"interval"`
	OneOff     bool              `json:"one-off"`
	Key        string            `json:"key,omitempty"`
	Cert       string            `json:"cert,omitempty"`
	Cacert     string            `json:"cacert,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// GetLabels returns the labels
func (h *HTTPHealthcheck) GetLabels() map[string]string {
	return h.Config.Labels
}

// Validate validates the healthcheck configuration
func (config *HTTPHealthcheckConfiguration) Validate() error {
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
	if !((config.Key != "" && config.Cert != "") ||
		(config.Key == "" && config.Cert == "")) {
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

// Summary returns an healthcheck summary
func (h *HTTPHealthcheck) Summary() string {
	summary := ""
	if h.Config.Description != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.Description, h.Config.Target, h.Config.Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.Target, h.Config.Port)
	}

	return summary
}

// Initialize the healthcheck.
func (h *HTTPHealthcheck) Initialize() error {
	h.buildURL()
	// tls is enabled
	dialer := net.Dialer{}
	tlsConfig := &tls.Config{}
	if h.Config.SourceIP != nil {
		srcIP := net.IP(h.Config.SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
		}
	}
	if h.Config.Key != "" {
		cert, err := tls.LoadX509KeyPair(h.Config.Cert, h.Config.Key)
		if err != nil {
			return errors.Wrapf(err, "Fail to load certificates")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if h.Config.Cacert != "" {
		caCert, err := ioutil.ReadFile(h.Config.Cacert)
		if err != nil {
			return errors.Wrapf(err, "Fail to load the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		result := caCertPool.AppendCertsFromPEM(caCert)
		if !result {
			return fmt.Errorf("fail to read ca certificate for healthcheck %s", h.Config.Name)
		}
		tlsConfig.RootCAs = caCertPool

	}
	tlsConfig.InsecureSkipVerify = h.Config.Insecure
	h.transport = &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: tlsConfig,
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

// GetConfig get the config
func (h *HTTPHealthcheck) GetConfig() interface{} {
	return h.Config
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
	ctx := h.t.Context(context.TODO())
	body := bytes.NewBuffer([]byte(h.Config.Body))
	req, err := http.NewRequest("GET", h.URL, body)
	if err != nil {
		return errors.Wrapf(err, "fail to initialize HTTP request")
	}
	req.Header.Set("User-Agent", "Cabourotte")
	for k, v := range h.Config.Headers {
		req.Header.Set(k, v)
	}
	redirect := http.ErrUseLastResponse
	if h.Config.Redirect {
		redirect = nil
	}
	client := &http.Client{
		Transport: h.transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return redirect
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
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrapf(err, "Fail to read request body")
	}
	responseBodyStr := string(responseBody)
	if !h.isSuccessful(response) {
		errorMsg := fmt.Sprintf("HTTP request failed: %d %s", response.StatusCode, html.EscapeString(responseBodyStr))
		err = errors.New(errorMsg)
		return err
	}
	for _, regex := range h.Config.BodyRegexp {
		r := regexp.Regexp(regex)
		if !r.MatchString(responseBodyStr) {
			return fmt.Errorf("healthcheck body does not match regex %s: %s", r.String(), responseBodyStr)
		}
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
