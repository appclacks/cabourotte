package riemanngo

import (
	"time"
)

// Event is a wrapper for Riemann events
type Event struct {
	TTL         time.Duration
	Time        time.Time
	Tags        []string
	Host        string
	State       string
	Service     string
	Metric      interface{} // Could be Int, Float32, Float64
	Description string
	Attributes  map[string]string
}
