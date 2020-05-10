package healthcheck

// Healthcheck is the face for an healthcheck
type Healthcheck interface {
	Initialize() error
	Start() error
	Stop() error
	Execute() error
	LogDebug(err error, message string)
	LogError(message string)
}
