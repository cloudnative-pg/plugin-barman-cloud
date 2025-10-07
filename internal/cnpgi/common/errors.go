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
