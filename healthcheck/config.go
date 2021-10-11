package healthcheck

import (
	"time"

	"github.com/pkg/errors"
)

type ConfigInterface interface {
	GetName() string
	GetInterval() Duration
	GetDescription() string
	GetOneOff() bool
	GetTimeout() Duration
	GetLabels() map[string]string
}

type BaseConfig struct {
	Name        string            `json:"name"`
	Interval    Duration          `json:"interval"`
	Description string            `json:"description"`
	OneOff      bool              `json:"one-off"`
	Timeout     Duration          `json:"timeout"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func (o BaseConfig) GetName() string {
	return o.Name
}

func (o BaseConfig) GetInterval() Duration {
	return o.Interval
}

func (o BaseConfig) GetDescription() string {
	return o.Description
}

func (o BaseConfig) GetOneOff() bool {
	return o.OneOff
}

func (o BaseConfig) GetTimeout() Duration {
	return o.Timeout
}

func (o BaseConfig) GetLabels() map[string]string {
	return o.Labels
}

func (o BaseConfig) Validate() error {
	if o.Name == "" {
		return errors.New("The healthcheck name is missing")
	}
	if o.Timeout == 0 {
		return errors.New("The healthcheck timeout is missing")
	}
	if !o.OneOff {
		if o.Interval < Duration(2*time.Second) {
			return errors.New("The healthcheck interval should be greater than 2 second")
		}
		if o.Interval < o.Timeout {
			return errors.New("The healthcheck interval should be greater than the timeout")
		}
	}
	return nil
}
