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
	"fmt"

	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("classifyWALRestoreError", func() {
	const walName = "000000010000000000000001"

	DescribeTable(
		"maps barman restorer sentinels to gRPC status codes",
		func(walErr error, expectedCode codes.Code) {
			got := classifyWALRestoreError(walName, walErr)
			Expect(got).To(HaveOccurred())

			st, ok := status.FromError(got)
			Expect(ok).To(BeTrue(), "returned error must carry a gRPC status")
			Expect(st.Code()).To(Equal(expectedCode))
			Expect(st.Message()).To(ContainSubstring(walName))
		},
		Entry("ErrWALNotFound -> NotFound",
			fmt.Errorf("object storage or file not found: %w", barmanRestorer.ErrWALNotFound),
			codes.NotFound),
		Entry("ErrInvalidWALName -> InvalidArgument",
			fmt.Errorf("invalid name for a WAL file: %w", barmanRestorer.ErrInvalidWALName),
			codes.InvalidArgument),
		Entry("ErrConnectivity -> Unavailable",
			fmt.Errorf("connectivity failure, retrying: %w", barmanRestorer.ErrConnectivity),
			codes.Unavailable),
		Entry("ErrGeneric -> Unavailable (barman uses exit 4 for some retryable cases too)",
			fmt.Errorf("generic error: %w", barmanRestorer.ErrGeneric),
			codes.Unavailable),
		Entry("unknown error -> Internal",
			errors.New("something we did not classify"),
			codes.Internal),
	)

	It("matches the sentinel even through several wrapping layers", func() {
		// The plugin wraps barman errors via fmt.Errorf("...: %w", ...);
		// classification must keep working if more wraps appear above.
		inner := fmt.Errorf("connectivity failure, retrying: %w", barmanRestorer.ErrConnectivity)
		wrapped := fmt.Errorf("while restoring WAL %s: %w", walName, inner)

		got := classifyWALRestoreError(walName, wrapped)
		st, ok := status.FromError(got)
		Expect(ok).To(BeTrue())
		Expect(st.Code()).To(Equal(codes.Unavailable))
	})

	It("treats ErrWALNotFound as terminal even when the error chain mentions other sentinels in its message", func() {
		// Defensive: if the underlying error stringifies to something
		// resembling another sentinel's message, the switch must still
		// match by identity (errors.Is), not by substring.
		walErr := fmt.Errorf("not found, looks like a connectivity failure: %w", barmanRestorer.ErrWALNotFound)

		got := classifyWALRestoreError(walName, walErr)
		st, _ := status.FromError(got)
		Expect(st.Code()).To(Equal(codes.NotFound))
	})
})
