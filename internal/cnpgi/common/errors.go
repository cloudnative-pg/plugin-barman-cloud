package common

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrEndOfWALStreamReached is returned when end of WAL is detected in the cloud archive.
var ErrEndOfWALStreamReached = status.Errorf(codes.OutOfRange, "end of WAL reached")

// ErrMissingPermissions is raised when the sidecar has no
// permission to download the credentials needed to reach
// the object storage.
// This will be fixed by the reconciliation loop in the
// operator plugin.
var ErrMissingPermissions = status.Errorf(codes.FailedPrecondition,
	"no permission to download the backup credentials, retrying")

// newWALNotFoundError returns a error that states that a
// certain WAL file has not been found. This error is
// compatible with GRPC status codes, resulting in a 404
// being used as a response code.
func newWALNotFoundError(walName string) error {
	return status.Errorf(codes.NotFound, "wal %q not found", walName)
}
