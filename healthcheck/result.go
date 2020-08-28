package healthcheck

import (
	"time"
)

// Result represents the result of an healthcheck
type Result struct {
	Name      string      `json:"name"`
	Summary   interface{} `json:"summary"`
	Success   bool        `json:"success"`
	Timestamp time.Time   `json:"timestamp"`
	Message   string      `json:"message"`
}

// NewResult build a a new result for an healthcheck
func NewResult(healthcheck Healthcheck, err error) *Result {
	now := time.Now()
	result := Result{
		Name:      healthcheck.Name(),
		Summary:   healthcheck.Summary(),
		Timestamp: now,
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
