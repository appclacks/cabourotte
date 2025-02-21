package healthcheck

import (
	"time"
)

// Result represents the result of an healthcheck
type Result struct {
	Name                 string            `json:"name"`
	Summary              interface{}       `json:"summary"`
	Labels               map[string]string `json:"labels,omitempty"`
	Success              bool              `json:"success"`
	HealthcheckTimestamp int64             `json:"healthcheck-timestamp"`
	Message              string            `json:"message"`
	MessageLabels        map[string]string `json:"message_labels"`
	Duration             int64             `json:"duration"`
	Source               string            `json:"source"`
}

// Equals implements Equals for Result
func (r Result) Equals(v Result) bool {
	if r.Name != v.Name {
		return false
	}
	if r.Summary != v.Summary {
		return false
	}
	if r.Success != v.Success {
		return false
	}
	if r.HealthcheckTimestamp != v.HealthcheckTimestamp {
		return false
	}
	if r.Message != v.Message {
		return false
	}
	if r.Duration != v.Duration {
		return false
	}
	if r.Source != v.Source {
		return false
	}
	if len(r.Labels) != len(v.Labels) {
		return false
	}
	for k, value := range r.Labels {
		if value != v.Labels[k] {
			return false
		}
	}
	return true
}

// NewResult build a a new result for an healthcheck
func NewResult(healthcheck Healthcheck, duration int64, labels map[string]string, err error) *Result {
	now := time.Now()
	source := "configuration"
	if healthcheck.Base().Source != "" {
		source = healthcheck.Base().Source
	}
	result := Result{
		Name:                 healthcheck.Base().Name,
		Summary:              healthcheck.Summary(),
		Labels:               healthcheck.Base().Labels,
		HealthcheckTimestamp: now.Unix(),
		Duration:             duration,
		Source:               source,
		MessageLabels:        labels,
	}
	if err != nil {
		result.Success = false
		result.Message = err.Error()
	} else {
		result.Success = true
		result.Message = "success"
	}
	return &result
}
