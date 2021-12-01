package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// TLSHealthcheckConfiguration defines a TLS healthcheck configuration
type TLSHealthcheckConfiguration struct {
	Base `json:",inline" yaml:",inline"`
	// can be an IP or a domain
	Target string `json:"target"`
	Port   uint   `json:"port"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	SourceIP IP `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	// +kubebuilder:validation:Type=string
	Timeout Duration `json:"timeout"`
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
	// +kubebuilder:validation:Optional
	Cert string `json:"cert,omitempty"`
	// +kubebuilder:validation:Optional
	Cacert string `json:"cacert,omitempty"`
	// +kubebuilder:validation:Optional
	ServerName string `json:"server-name,omitempty" yaml:"server-name"`
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	ExpirationDelay Duration `json:"expiration-delay" yaml:"expiration-delay"`
}

// TLSHealthcheck defines a TLS healthcheck
type TLSHealthcheck struct {
	Logger    *zap.Logger
	Config    *TLSHealthcheckConfiguration
	URL       string
	TLSConfig *tls.Config

	Tick *time.Ticker
	t    tomb.Tomb
}

// Validate validates the healthcheck configuration
func (config *TLSHealthcheckConfiguration) Validate() error {
	if config.Base.Name == "" {
		return errors.New("The healthcheck name is missing")
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

// Base get the base configuration
func (h *TLSHealthcheck) Base() Base {
	return h.Config.Base
}

// SetSource set the healthcheck source
func (h *TLSHealthcheck) SetSource(source string) {
	h.Config.Base.Source = source
}

// Summary returns an healthcheck summary
func (h *TLSHealthcheck) Summary() string {
	summary := ""
	if h.Config.Base.Description != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.Base.Description, h.Config.Target, h.Config.Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.Target, h.Config.Port)
	}

	return summary
}

// buildURL build the target URL for the TLS healthcheck, depending of its
// configuration
func (h *TLSHealthcheck) buildURL() {
	h.URL = net.JoinHostPort(h.Config.Target, fmt.Sprintf("%d", h.Config.Port))
}

// Initialize the healthcheck.
func (h *TLSHealthcheck) Initialize() error {
	h.buildURL()
	tlsConfig := &tls.Config{}
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
			return fmt.Errorf("fail to read ca certificate for healthcheck %s", h.Config.Base.Name)
		}
		tlsConfig.RootCAs = caCertPool
	}
	if h.Config.ServerName != "" {
		tlsConfig.ServerName = h.Config.ServerName
	} else {
		tlsConfig.ServerName = h.Config.Target
	}
	tlsConfig.InsecureSkipVerify = h.Config.Insecure
	h.TLSConfig = tlsConfig
	return nil
}

// GetConfig get the config
func (h *TLSHealthcheck) GetConfig() interface{} {
	return h.Config
}

// LogError logs an error with context
func (h *TLSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Base.Name))
}

// LogDebug logs a message with context
func (h *TLSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Base.Name))
}

// LogInfo logs a message with context
func (h *TLSHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Base.Name))
}

// Execute executes an healthcheck on the given target
func (h *TLSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	dialer := net.Dialer{}
	ctx := h.t.Context(context.TODO())
	if h.Config.SourceIP != nil {
		srcIP := net.IP(h.Config.SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
			Timeout:   time.Duration(h.Config.Timeout),
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.Timeout))
	defer cancel()
	conn, err := dialer.DialContext(timeoutCtx, "tcp", h.URL)
	if err != nil {
		return errors.Wrapf(err, "TLS connection failed on %s", h.URL)
	}
	defer conn.Close()
	tlsConn := tls.Client(conn, h.TLSConfig)
	defer tlsConn.Close()
	err = tlsConn.Handshake()
	if err != nil {
		return errors.Wrapf(err, "TLS handshake failed on %s", h.URL)
	}
	if h.Config.ExpirationDelay != 0 {
		state := tlsConn.ConnectionState()
		expirationTime := time.Time{}
		for _, cert := range state.PeerCertificates {
			if (expirationTime.IsZero() || cert.NotAfter.Before(expirationTime)) && !cert.NotAfter.IsZero() {
				expirationTime = cert.NotAfter
			}
		}
		expirationTimeLimit := time.Now().Add(time.Duration(h.Config.ExpirationDelay))
		if expirationTime.Before(expirationTimeLimit) {
			return fmt.Errorf("The certificate for %s will expire at %s", h.URL, expirationTime.String())
		}
	}

	return nil
}

// NewTLSHealthcheck creates a TLS healthcheck from a logger and a configuration
func NewTLSHealthcheck(logger *zap.Logger, config *TLSHealthcheckConfiguration) *TLSHealthcheck {
	return &TLSHealthcheck{
		Logger: logger,
		Config: config,
	}
}

// MarshalJSON marshal to json a dns healthcheck
func (h *TLSHealthcheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TLSHealthcheckConfiguration) DeepCopyInto(out *TLSHealthcheckConfiguration) {
	*out = *in
	in.Base.DeepCopyInto(&out.Base)
	if in.SourceIP != nil {
		in, out := &in.SourceIP, &out.SourceIP
		*out = make(IP, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TLSHealthcheckConfiguration.
func (in *TLSHealthcheckConfiguration) DeepCopy() *TLSHealthcheckConfiguration {
	if in == nil {
		return nil
	}
	out := new(TLSHealthcheckConfiguration)
	in.DeepCopyInto(out)
	return out
}
