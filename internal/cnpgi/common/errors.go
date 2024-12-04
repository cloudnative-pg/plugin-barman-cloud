package common

// WALNotFoundError is raised when a WAL file has not been found in the object store
type WALNotFoundError struct {
}

// ShouldPrintStackTrace tells whether the sidecar log stream should contain the stack trace
func (e WALNotFoundError) ShouldPrintStackTrace() bool {
	return false
}

// Error implements the error interface
func (e WALNotFoundError) Error() string {
	return "WAL file not found"
}
