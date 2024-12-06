package common

// walNotFoundError is raised when a WAL file has not been found in the object store
type walNotFoundError struct{}

func newWALNotFoundError() *walNotFoundError { return &walNotFoundError{} }

// ShouldPrintStackTrace tells whether the sidecar log stream should contain the stack trace
func (e walNotFoundError) ShouldPrintStackTrace() bool {
	return false
}

// Error implements the error interface
func (e walNotFoundError) Error() string {
	return "WAL file not found"
}
