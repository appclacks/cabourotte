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
	Name        string `json:"name"`
	Description string `json:"description"`
	// can be an IP or a domain
	Target          string   `json:"target"`
	Port            uint     `json:"port"`
	SourceIP        IP       `json:"source-ip" yaml:"source-ip"`
	Timeout         Duration `json:"timeout"`
	Interval        Duration `json:"interval"`
	OneOff          bool     `json:"one-off"`
	Key             string   `json:"key,omitempty"`
	Cert            string   `json:"cert,omitempty"`
	Cacert          string   `json:"cacert,omitempty"`
	ServerName      string   `json:"server-name,omitempty" yaml:"server-name"`
	Insecure        bool
	ExpirationDelay Duration          `json:"expiration-delay" yaml:"expiration-delay"`
	Labels          map[string]string `json:"labels,omitempty"`
}

// GetLabels returns the labels
func (h *TLSHealthcheck) GetLabels() map[string]string {
	return h.Config.Labels
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
	if config.Name == "" {
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

// Name returns the healthcheck identifier.
func (h *TLSHealthcheck) Name() string {
	return h.Config.Name
}

// Summary returns an healthcheck summary
func (h *TLSHealthcheck) Summary() string {
	summary := ""
	if h.Config.Description != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.Description, h.Config.Target, h.Config.Port)

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
		tlsConfig.Certificates = []tls.Certificate{cert}
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

// Interval Get the interval.
func (h *TLSHealthcheck) Interval() Duration {
	return h.Config.Interval
}

// GetConfig get the config
func (h *TLSHealthcheck) GetConfig() interface{} {
	return h.Config
}

// OneOff returns true if the healthcheck if a one-off check
func (h *TLSHealthcheck) OneOff() bool {
	return h.Config.OneOff

}

// LogError logs an error with context
func (h *TLSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogDebug logs a message with context
func (h *TLSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogInfo logs a message with context
func (h *TLSHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// Execute executes an healthcheck on the given target
func (h *TLSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	dialer := net.Dialer{}
	ctx := h.t.Context(nil)
	if h.Config.SourceIP != nil {
		srcIP := net.IP(h.Config.SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
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
