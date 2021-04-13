package healthcheck

import (
	"time"

	"gopkg.in/tomb.v2"
)

// Wrapper Wrap an healthcheck
type Wrapper struct {
	healthcheck Healthcheck
	Tick        *time.Ticker
	t           tomb.Tomb
}

// NewWrapper creates a new wrapper struct
func NewWrapper(healthcheck Healthcheck) *Wrapper {
	return &Wrapper{
		healthcheck: healthcheck,
	}
}

// Stop an Healthcheck wrapper
func (w *Wrapper) Stop() error {
	w.Tick.Stop()
	w.t.Kill(nil)
	err := w.t.Wait()
	if err != nil {
		return err
	}
	return nil

}
