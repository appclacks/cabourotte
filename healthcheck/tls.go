package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	BaseConfig `json:",inline"`
	// can be an IP or a domain
	Target          string `json:"target"`
	Port            uint   `json:"port"`
	SourceIP        IP     `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	Key             string `json:"key,omitempty"`
	Cert            string `json:"cert,omitempty"`
	Cacert          string `json:"cacert,omitempty"`
	ServerName      string `json:"server-name,omitempty" yaml:"server-name"`
	Insecure        bool
	ExpirationDelay Duration `json:"expiration-delay" yaml:"expiration-delay"`
}

// TLSHealthcheck defines a TLS healthcheck
type TLSHealthcheck struct {
	Base
	TLSConfig *tls.Config
	t         tomb.Tomb
}

// Validate validates the healthcheck configuration
func (config *TLSHealthcheckConfiguration) Validate() error {
	if err := config.BaseConfig.Validate(); err != nil {
		return err
	}
	if config.Target == "" {
		return errors.New("The healthcheck target is missing")
	}
	if config.Port == 0 {
		return errors.New("The healthcheck port is missing")
	}
	if !((config.Key != "" && config.Cert != "") ||
		(config.Key == "" && config.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	return nil
}

// Summary returns an healthcheck summary
func (h *TLSHealthcheck) Summary() string {
	summary := ""
	if h.Config.GetDescription() != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.GetDescription(), h.Config.(*TLSHealthcheckConfiguration).Target, h.Config.(*TLSHealthcheckConfiguration).Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.(*TLSHealthcheckConfiguration).Target, h.Config.(*TLSHealthcheckConfiguration).Port)
	}

	return summary
}

// buildURL build the target URL for the TLS healthcheck, depending of its
// configuration
func (h *TLSHealthcheck) buildURL() {
	h.URL = net.JoinHostPort(h.Config.(*TLSHealthcheckConfiguration).Target, fmt.Sprintf("%d", h.Config.(*TLSHealthcheckConfiguration).Port))
}

// Initialize the healthcheck.
func (h *TLSHealthcheck) Initialize() error {
	h.buildURL()
	tlsConfig := &tls.Config{}
	if h.Config.(*TLSHealthcheckConfiguration).Key != "" {
		cert, err := tls.LoadX509KeyPair(h.Config.(*TLSHealthcheckConfiguration).Cert, h.Config.(*TLSHealthcheckConfiguration).Key)
		if err != nil {
			return errors.Wrapf(err, "Fail to load certificates")
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if h.Config.(*TLSHealthcheckConfiguration).Cacert != "" {
		caCert, err := ioutil.ReadFile(h.Config.(*TLSHealthcheckConfiguration).Cacert)
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
	if h.Config.(*TLSHealthcheckConfiguration).ServerName != "" {
		tlsConfig.ServerName = h.Config.(*TLSHealthcheckConfiguration).ServerName
	} else {
		tlsConfig.ServerName = h.Config.(*TLSHealthcheckConfiguration).Target
	}
	tlsConfig.InsecureSkipVerify = h.Config.(*TLSHealthcheckConfiguration).Insecure
	h.TLSConfig = tlsConfig
	return nil
}

// LogError logs an error with context
func (h *TLSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.(*TLSHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TLSHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogDebug logs a message with context
func (h *TLSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.(*TLSHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TLSHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogInfo logs a message with context
func (h *TLSHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.(*TLSHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TLSHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// Execute executes an healthcheck on the given target
func (h *TLSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	dialer := net.Dialer{}
	ctx := h.t.Context(context.TODO())
	if h.Config.(*TLSHealthcheckConfiguration).SourceIP != nil {
		srcIP := net.IP(h.Config.(*TLSHealthcheckConfiguration).SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
			Timeout:   time.Duration(h.Config.GetTimeout()),
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.GetTimeout()))
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
	if h.Config.(*TLSHealthcheckConfiguration).ExpirationDelay != 0 {
		state := tlsConn.ConnectionState()
		expirationTime := time.Time{}
		for _, cert := range state.PeerCertificates {
			if (expirationTime.IsZero() || cert.NotAfter.Before(expirationTime)) && !cert.NotAfter.IsZero() {
				expirationTime = cert.NotAfter
			}
		}
		expirationTimeLimit := time.Now().Add(time.Duration(h.Config.(*TLSHealthcheckConfiguration).ExpirationDelay))
		if expirationTime.Before(expirationTimeLimit) {
			return fmt.Errorf("The certificate for %s will expire at %s", h.URL, expirationTime.String())
		}
	}

	return nil
}

// NewTLSHealthcheck creates a TLS healthcheck from a logger and a configuration
func NewTLSHealthcheck(logger *zap.Logger, config *TLSHealthcheckConfiguration) *TLSHealthcheck {
	return &TLSHealthcheck{
		Base: Base{
			Logger: logger,
			Config: config,
		},
	}
}
