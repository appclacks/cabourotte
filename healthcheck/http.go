package healthcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"time"

	"github.com/appclacks/cabourotte/tls"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// HTTPHealthcheckConfiguration defines an HTTP healthcheck configuration
type HTTPHealthcheckConfiguration struct {
	Base        `json:",inline" yaml:",inline"`
	ValidStatus []uint `json:"valid-status" yaml:"valid-status"`
	// can be an IP or a domain
	Target     string            `json:"target"`
	Host       string            `json:"host,omitempty"`
	Method     string            `json:"method"`
	Port       uint              `json:"port"`
	Redirect   bool              `json:"redirect"`
	Body       string            `json:"body,omitempty"`
	Query      map[string]string `json:"query,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Protocol   Protocol          `json:"protocol"`
	Path       string            `json:"path,omitempty"`
	SourceIP   IP                `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	BodyRegexp []Regexp          `json:"body-regexp,omitempty" yaml:"body-regexp,omitempty"`
	Insecure   bool              `json:"insecure"`
	ServerName string            `json:"server-name"`
	Timeout    Duration          `json:"timeout"`
	Key        string            `json:"key,omitempty"`
	Cert       string            `json:"cert,omitempty"`
	Cacert     string            `json:"cacert,omitempty"`
}

// Validate validates the healthcheck configuration
func (config *HTTPHealthcheckConfiguration) Validate() error {
	if config.Base.Name == "" {
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
	if config.Method != "" {
		if config.Method != "GET" && config.Method != "POST" && config.Method != "PUT" && config.Method != "HEAD" && config.Method != "DELETE" {
			return errors.New(fmt.Sprintf("The healthcheck method is invalid: %s", config.Method))
		}
	} else {
		config.Method = "GET"
	}
	if !config.Base.OneOff {
		if config.Base.Interval < Duration(2*time.Second) {
			return errors.New("The healthcheck interval should be greater than 2 second")
		}
		if config.Base.Interval < config.Timeout {
			return errors.New("The healthcheck interval should be greater than the timeout")
		}
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

	Tick   *time.Ticker
	Client *http.Client
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

// Summary returns an healthcheck summary
func (h *HTTPHealthcheck) Summary() string {
	summary := ""
	if h.Config.Base.Description != "" {
		summary = fmt.Sprintf("HTTP healthcheck %s on %s:%d", h.Config.Base.Description, h.Config.Target, h.Config.Port)

	} else {
		summary = fmt.Sprintf("HTTP healthcheck on %s:%d", h.Config.Target, h.Config.Port)
	}

	return summary
}

// Initialize the healthcheck.
func (h *HTTPHealthcheck) Initialize() error {
	h.buildURL()

	dialer := net.Dialer{}
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
	tlsConfig, err := tls.GetTLSConfig(h.Config.Key, h.Config.Cert, h.Config.Cacert, h.Config.ServerName, h.Config.Insecure)
	if err != nil {
		return err
	}
	transport := &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: tlsConfig,
	}
	redirect := http.ErrUseLastResponse
	if h.Config.Redirect {
		redirect = nil
	}
	h.Client = &http.Client{
		Transport: otelhttp.NewTransport(
			transport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}),
		),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return redirect
		},
	}
	return nil
}

// GetConfig get the config
func (h *HTTPHealthcheck) GetConfig() interface{} {
	return h.Config
}

// Base get the base configuration
func (h *HTTPHealthcheck) Base() Base {
	return h.Config.Base
}

// SetSource set the healthcheck source
func (h *HTTPHealthcheck) SetSource(source string) {
	h.Config.Base.Source = source
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
		zap.String("name", h.Config.Base.Name))
}

// LogDebug logs a message with context
func (h *HTTPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Base.Name))
}

// LogInfo logs a message with context
func (h *HTTPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Base.Name))
}

// Execute executes an healthcheck on the given target
func (h *HTTPHealthcheck) Execute(ctx *context.Context) error {
	h.LogDebug("start executing healthcheck")
	body := bytes.NewBuffer([]byte(h.Config.Body))
	req, err := http.NewRequest(h.Config.Method, h.URL, body)
	if err != nil {
		return errors.Wrapf(err, "fail to initialize HTTP request")
	}
	if h.Config.Host != "" {
		req.Host = h.Config.Host
	}
	req.Header.Set("User-Agent", "Cabourotte")
	for k, v := range h.Config.Headers {
		req.Header.Set(k, v)
	}
	if h.Config.Host != "" {
		req.Host = h.Config.Host
	}
	client := h.Client
	timeoutCtx, cancel := context.WithTimeout(*ctx, time.Duration(h.Config.Timeout))
	defer cancel()
	req = req.WithContext(timeoutCtx)
	if len(h.Config.Query) != 0 {
		q := req.URL.Query()
		for k, v := range h.Config.Query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	response, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "HTTP request failed")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Wrapf(err, "Fail to read request body")
	}
	responseBodyStr := string(responseBody)
	maxMessageSize := 1000
	message := responseBodyStr
	if len(responseBodyStr) > maxMessageSize {
		message = responseBodyStr[0:maxMessageSize]
	}
	*ctx = context.WithValue(*ctx, "labels", map[string]string{"HTTP Status Code": fmt.Sprintf("%v", response.StatusCode)})
	if !h.isSuccessful(response) {
		errorMsg := fmt.Sprintf("HTTP request failed: status %d. Body: '%s'", response.StatusCode, html.EscapeString(message))
		err = errors.New(errorMsg)
		return err
	}
	for _, regex := range h.Config.BodyRegexp {
		r := regexp.Regexp(regex)
		if !r.MatchString(responseBodyStr) {
			return fmt.Errorf("healthcheck body does not match regex %s: %s", r.String(), message)
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPHealthcheckConfiguration) DeepCopyInto(out *HTTPHealthcheckConfiguration) {
	*out = *in
	in.Base.DeepCopyInto(&out.Base)
	if in.ValidStatus != nil {
		in, out := &in.ValidStatus, &out.ValidStatus
		*out = make([]uint, len(*in))
		copy(*out, *in)
	}
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SourceIP != nil {
		in, out := &in.SourceIP, &out.SourceIP
		*out = make(IP, len(*in))
		copy(*out, *in)
	}
	if in.BodyRegexp != nil {
		in, out := &in.BodyRegexp, &out.BodyRegexp
		*out = make([]Regexp, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPHealthcheckConfiguration.
func (in *HTTPHealthcheckConfiguration) DeepCopy() *HTTPHealthcheckConfiguration {
	if in == nil {
		return nil
	}
	out := new(HTTPHealthcheckConfiguration)
	in.DeepCopyInto(out)
	return out
}
