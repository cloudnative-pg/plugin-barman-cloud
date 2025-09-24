package common

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newWALNotFoundError returns a error that states that a
// certain WAL file has not been found. This error is
// compatible with GRPC status codes, resulting in a 404
// being used as a response code.
func newWALNotFoundError(walName string) error {
	return status.Errorf(codes.NotFound, "wal %q not found", walName)
}
