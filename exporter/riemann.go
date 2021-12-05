package exporter

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/riemann/riemann-go-client"
	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/tls"
)

// RiemannConfiguration the Riemann exporter configuration
type RiemannConfiguration struct {
	Name     string
	Host     string
	Port     uint32
	TTL      healthcheck.Duration
	Key      string `json:"key,omitempty"`
	Cert     string `json:"cert,omitempty"`
	Cacert   string `json:"cacert,omitempty"`
	Insecure bool
}

// RiemannExporter the Riemann exporter struct
type RiemannExporter struct {
	Started bool
	Logger  *zap.Logger
	Config  *RiemannConfiguration
	Client  riemanngo.Client
}

// UnmarshalYAML parses the configuration of the Riemann component from YAML.
func (c *RiemannConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration RiemannConfiguration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Riemann exporter configuration")
	}
	if raw.Host == "" {
		return errors.New("Invalid host for the Riemann exporter configuration")
	}
	if raw.Name == "" {
		return errors.New("Invalid name for the Riemann exporter configuration")
	}
	if raw.Port == 0 {
		return errors.New("Invalid port for the Riemann server")
	}
	if !((raw.Key != "" && raw.Cert != "") ||
		(raw.Key == "" && raw.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	if raw.TTL == 0 {
		raw.TTL = healthcheck.Duration(time.Second * 60)
	}
	*c = RiemannConfiguration(raw)
	return nil
}

func getClient(config *RiemannConfiguration) (riemanngo.Client, error) {
	var client riemanngo.Client
	url := net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))
	if config.Key != "" || config.Cert != "" || config.Cacert != "" {
		tlsConfig, err := tls.GetTLSConfig(config.Key, config.Cert, config.Cacert, config.Insecure)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to build the Riemann exporter tls configuration")
		}
		client, err = riemanngo.NewTLSClient(url, tlsConfig, 5*time.Second)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to build the Riemann tls client")
		}

	} else {
		client = riemanngo.NewTCPClient(url, 5*time.Second)
	}
	return client, nil
}

// NewRiemannExporter creates a new Riemann exporter from the configuration
func NewRiemannExporter(logger *zap.Logger, config *RiemannConfiguration) (*RiemannExporter, error) {
	client, err := getClient(config)
	if err != nil {
		return nil, err
	}
	exporter := &RiemannExporter{
		Client: client,
		Logger: logger,
		Config: config,
	}
	return exporter, nil
}

// Start starts the Riemann exporter component
func (c *RiemannExporter) Start() error {
	// nothing to do
	c.Logger.Info(fmt.Sprintf("Starting the Riemann healthcheck exporter on %s:%d", c.Config.Host, c.Config.Port))
	err := c.Client.Connect()
	if err != nil {
		return errors.Wrapf(err, "Fail to start the Riemann exporter")
	}
	c.Started = true
	return nil
}

// Stop stops the Riemann exporter component
func (c *RiemannExporter) Stop() error {
	c.Logger.Info(fmt.Sprintf("Stopping the Riemann exporter %s", c.Config.Name))
	c.Started = false
	return c.Client.Close()
}

// Reconnect reconnects the Riemann exporter component
func (c *RiemannExporter) Reconnect() error {
	c.Logger.Info("Riemann exporter: reconnecting")
	client, err := getClient(c.Config)
	if err != nil {
		return err
	}
	c.Client = client
	err = c.Client.Connect()
	if err != nil {
		return errors.Wrapf(err, "Fail to restart the Riemann exporter")
	}
	c.Logger.Info("Riemann exporter: reconnected")
	c.Started = true
	return nil
}

// Name returns the name of the exporter
func (c *RiemannExporter) Name() string {
	return c.Config.Name
}

// GetConfig returns the config of the exporter
func (c *RiemannExporter) GetConfig() interface{} {
	return c.Config
}

// IsStarted returns the exporter status
func (c *RiemannExporter) IsStarted() bool {
	return c.Started
}

// Push pushes events to the desination
func (c *RiemannExporter) Push(result *healthcheck.Result) error {
	state := "ok"
	if !result.Success {
		state = "critical"
	}
	attributes := map[string]string{
		"healthcheck": result.Name,
		"source":      result.Source,
	}
	for k, v := range result.Labels {
		attributes[k] = v
	}
	event := &riemanngo.Event{
		Service:     "cabourotte-healthcheck",
		Metric:      result.Duration,
		Description: fmt.Sprintf("%s: %s", result.Summary, result.Message),
		Time:        time.Unix(result.HealthcheckTimestamp, 0),
		State:       state,
		Tags:        []string{"cabourotte"},
		TTL:         time.Duration(c.Config.TTL),
		Attributes:  attributes,
	}
	response, err := riemanngo.SendEvent(c.Client, event)
	if err != nil {
		return errors.Wrapf(err, "Riemann exporter: fail to send event")
	}
	if !*response.Ok {
		c.Logger.Info(fmt.Sprintf("Riemann returned an error in the exporter %s: %s", c.Config.Name, *response.Error))
	}
	return nil
}
