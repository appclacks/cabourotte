package healthcheck

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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
	BaseConfig  `json:",inline"`
	ValidStatus []uint `json:"valid-status" yaml:"valid-status"`
	// can be an IP or a domain
	Target     string            `json:"target"`
	Method     string            `json:"method"`
	Port       uint              `json:"port"`
	Redirect   bool              `json:"redirect"`
	Body       string            `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Protocol   Protocol          `json:"protocol"`
	Path       string            `json:"path,omitempty"`
	SourceIP   IP                `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	BodyRegexp []Regexp          `json:"body-regexp,omitempty" yaml:"body-regexp,omitempty"`
	Insecure   bool
	Key        string `json:"key,omitempty"`
	Cert       string `json:"cert,omitempty"`
	Cacert     string `json:"cacert,omitempty"`
}

// Validate validates the healthcheck configuration
func (config *HTTPHealthcheckConfiguration) Validate() error {
	if err := config.BaseConfig.Validate(); err != nil {
		return err
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
	if config.Method != "" {
		if config.Method != "GET" && config.Method != "POST" && config.Method != "PUT" && config.Method != "HEAD" {
			return errors.New(fmt.Sprintf("The healthcheck method is invalid: %s", config.Method))
		}
	} else {
		config.Method = "GET"
	}
	if !((config.Key != "" && config.Cert != "") ||
		(config.Key == "" && config.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	return nil
}

// HTTPHealthcheck defines an HTTP healthcheck
type HTTPHealthcheck struct {
	Base
	t         tomb.Tomb
	transport *http.Transport
}

// buildURL build the target URL for the HTTP healthcheck, depending of its
// configuration
func (h *HTTPHealthcheck) buildURL() {
	protocol := "http"
	if h.Config.(*HTTPHealthcheckConfiguration).Protocol == HTTPS {
		protocol = "https"
	}
	h.URL = fmt.Sprintf(
		"%s://%s%s",
		protocol,
		net.JoinHostPort(h.Config.(*HTTPHealthcheckConfiguration).Target, fmt.Sprintf("%d", h.Config.(*HTTPHealthcheckConfiguration).Port)),
		h.Config.(*HTTPHealthcheckConfiguration).Path)
}

// Summary returns an healthcheck summary
func (h *HTTPHealthcheck) Summary() string {
	summary := ""
	if h.Config.GetDescription() != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.GetDescription(), h.Config.(*HTTPHealthcheckConfiguration).Target, h.Config.(*HTTPHealthcheckConfiguration).Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.(*HTTPHealthcheckConfiguration).Target, h.Config.(*HTTPHealthcheckConfiguration).Port)
	}

	return summary
}

// Initialize the healthcheck.
func (h *HTTPHealthcheck) Initialize() error {
	h.buildURL()
	// tls is enabled
	dialer := net.Dialer{}
	tlsConfig := &tls.Config{}
	if h.Config.(*HTTPHealthcheckConfiguration).SourceIP != nil {
		srcIP := net.IP(h.Config.(*HTTPHealthcheckConfiguration).SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
		}
	}
	if h.Config.(*HTTPHealthcheckConfiguration).Key != "" {
		cert, err := tls.LoadX509KeyPair(h.Config.(*HTTPHealthcheckConfiguration).Cert, h.Config.(*HTTPHealthcheckConfiguration).Key)
		if err != nil {
			return errors.Wrapf(err, "Fail to load certificates")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if h.Config.(*HTTPHealthcheckConfiguration).Cacert != "" {
		caCert, err := ioutil.ReadFile(h.Config.(*HTTPHealthcheckConfiguration).Cacert)
		if err != nil {
			return errors.Wrapf(err, "Fail to load the ca certificate")
		}
		caCertPool := x509.NewCertPool()
		result := caCertPool.AppendCertsFromPEM(caCert)
		if !result {
			return fmt.Errorf("fail to read ca certificate for healthcheck %s", h.Config.GetName())
		}
		tlsConfig.RootCAs = caCertPool

	}
	tlsConfig.InsecureSkipVerify = h.Config.(*HTTPHealthcheckConfiguration).Insecure
	h.transport = &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: tlsConfig,
	}
	return nil
}

// isSuccessful verifies if a healthcheck result is considered valid
// depending of the healthcheck configuration
func (h *HTTPHealthcheck) isSuccessful(response *http.Response) bool {
	for _, s := range h.Config.(*HTTPHealthcheckConfiguration).ValidStatus {
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
		zap.String("target", h.Config.(*HTTPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*HTTPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogDebug logs a message with context
func (h *HTTPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.(*HTTPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*HTTPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogInfo logs a message with context
func (h *HTTPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.(*HTTPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*HTTPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// Execute executes an healthcheck on the given target
func (h *HTTPHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx := h.t.Context(context.TODO())
	body := bytes.NewBuffer([]byte(h.Config.(*HTTPHealthcheckConfiguration).Body))
	req, err := http.NewRequest(h.Config.(*HTTPHealthcheckConfiguration).Method, h.URL, body)
	if err != nil {
		return errors.Wrapf(err, "fail to initialize HTTP request")
	}
	req.Header.Set("User-Agent", "Cabourotte")
	for k, v := range h.Config.(*HTTPHealthcheckConfiguration).Headers {
		req.Header.Set(k, v)
	}
	redirect := http.ErrUseLastResponse
	if h.Config.(*HTTPHealthcheckConfiguration).Redirect {
		redirect = nil
	}
	client := &http.Client{
		Transport: h.transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return redirect
		},
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.(*HTTPHealthcheckConfiguration).Timeout))
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
	for _, regex := range h.Config.(*HTTPHealthcheckConfiguration).BodyRegexp {
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
		Base: Base{
			Logger: logger,
			Config: config,
		},
	}
}
