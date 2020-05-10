package healthcheck

// Healthcheck is the face for an healthcheck
type Healthcheck interface {
	Start() error
	Stop() error
	Execute() error
}
