/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"errors"

	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// classifyWALRestoreError maps an error returned by the WAL
// restorer to a gRPC-coded error so the caller can tell terminal
// failures apart from transient ones via the status code.
func classifyWALRestoreError(walName string, walErr error) error {
	switch {
	case errors.Is(walErr, barmanRestorer.ErrWALNotFound):
		return newWALNotFoundError(walName)
	case errors.Is(walErr, barmanRestorer.ErrInvalidWALName):
		// A malformed WAL name will never become valid on retry.
		return newInvalidWALNameError(walName, walErr)
	case errors.Is(walErr, barmanRestorer.ErrConnectivity),
		errors.Is(walErr, barmanRestorer.ErrGeneric):
		// barman-cloud exit codes 2 (connectivity) and 4
		// (generic) both surface conditions that are retryable
		// in practice — barman uses the "generic" bucket for
		// some connection-class failures too, not just exit 2.
		return newUnavailableError(walName, walErr)
	default:
		// Unrecognized exit codes and unexpected failures (e.g.
		// the barman-cloud command could not be executed). No
		// positive signal that retry would help.
		return newInternalWALRestoreError(walName, walErr)
	}
}

// ErrEndOfWALStreamReached is returned when end of WAL is detected in the cloud archive.
var ErrEndOfWALStreamReached = status.Errorf(codes.OutOfRange, "end of WAL reached")

// ErrMissingPermissions is raised when the sidecar has no
// permission to download the credentials needed to reach
// the object storage.
// This will be fixed by the reconciliation loop in the
// operator plugin.
var ErrMissingPermissions = status.Errorf(codes.FailedPrecondition,
	"no permission to download the backup credentials, retrying")

// newWALNotFoundError reports that the requested WAL file is not
// present in the object store. Emits codes.NotFound: this is a
// terminal condition (the file won't appear on retry).
func newWALNotFoundError(walName string) error {
	return status.Errorf(codes.NotFound, "wal %q not found", walName)
}

// newUnavailableError reports that downloading the WAL file failed
// for a reason expected to be transient (a barman-cloud connectivity
// blip, or a generic exit code that in practice covers retryable
// conditions too). Emits codes.Unavailable: per gRPC conventions,
// the canonical signal for "retry may succeed".
func newUnavailableError(walName string, err error) error {
	return status.Errorf(
		codes.Unavailable,
		"transient error while downloading %q: %s",
		walName,
		err.Error(),
	)
}

// newInvalidWALNameError reports that the requested WAL name is
// not a valid name. Emits codes.InvalidArgument: this is a
// terminal condition (the same name won't become valid on retry).
func newInvalidWALNameError(walName string, err error) error {
	return status.Errorf(
		codes.InvalidArgument,
		"invalid WAL name %q: %s",
		walName,
		err.Error(),
	)
}

// newInternalWALRestoreError reports that downloading the WAL
// file failed for an unclassified reason. Emits codes.Internal:
// we have no positive signal that retry would help, so by gRPC
// convention this is treated as terminal.
func newInternalWALRestoreError(walName string, err error) error {
	return status.Errorf(
		codes.Internal,
		"internal error while downloading %q: %s",
		walName,
		err.Error(),
	)
}
