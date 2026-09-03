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

package restore

import (
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resolveRestoreEmptyWalArchiveCheck", func() {
	// skipAnnotation mirrors the unexported constant in cloudnative-pg's
	// pkg/utils; hard-coding the literal makes a divergence surface as a
	// failing test rather than silently disabling the check.
	const skipAnnotation = "cnpg.io/skipEmptyWalArchiveCheck"

	clusterWith := func(annotationValue *string) *cnpgv1.Cluster {
		cluster := &cnpgv1.Cluster{}
		if annotationValue != nil {
			cluster.Annotations = map[string]string{skipAnnotation: *annotationValue}
		}
		return cluster
	}

	When("the operator sets the decision", func() {
		It("obeys true even when the annotation would skip the check", func() {
			Expect(resolveRestoreEmptyWalArchiveCheck(ptr.To(true), clusterWith(ptr.To("enabled")))).To(BeTrue())
		})

		It("obeys false even when the annotation would keep the check on", func() {
			Expect(resolveRestoreEmptyWalArchiveCheck(ptr.To(false), clusterWith(nil))).To(BeFalse())
		})
	})

	When("the operator predates the field (nil decision)", func() {
		DescribeTable(
			"falls back to the Cluster annotation, never to a marker file",
			func(annotationValue *string, expected bool) {
				Expect(resolveRestoreEmptyWalArchiveCheck(nil, clusterWith(annotationValue))).To(Equal(expected))
			},
			Entry("no annotation: check runs", nil, true),
			Entry("opt-out annotation: check skipped", ptr.To("enabled"), false),
			Entry("unrelated annotation value: check runs", ptr.To("something-else"), true),
			Entry("empty annotation value: check runs", ptr.To(""), true),
		)
	})
})
