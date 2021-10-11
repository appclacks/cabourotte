package healthcheck

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gopkg.in/tomb.v2"
)

type Base struct {
	Logger *zap.Logger
	Config ConfigInterface
	URL    string
	Tick   *time.Ticker
	t      tomb.Tomb
}

func (o *Base) GetTick() *time.Ticker {
	return o.Tick
}

func (o *Base) SetTick(tick *time.Ticker) {
	o.Tick = tick
}

func (o *Base) GetT() *tomb.Tomb {
	return &o.t
}

// Interval Get the interval.
func (o *Base) Interval() Duration {
	return o.Config.GetInterval()
}

// GetConfig get the config
func (o *Base) GetConfig() interface{} {
	return o.Config
}

// Name returns the healthcheck identifier.
func (o *Base) Name() string {
	return o.Config.GetName()
}

func (o *Base) OneOff() bool {
	return o.Config.GetOneOff()
}

func (o *Base) GetLabels() map[string]string {
	return o.Config.GetLabels()
}

// MarshalJSON marshal to json a dns healthcheck
func (h *Base) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}

// Stop an Healthcheck wrapper
func (w *Base) Stop() error {
	w.Tick.Stop()
	w.t.Kill(nil)
	err := w.t.Wait()
	if err != nil {
		return err
	}
	return nil

}
