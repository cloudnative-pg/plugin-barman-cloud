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

package instance

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// almostOne is the largest value returned by rand.Float64
const almostOne = 1 - 1e-12

func constRand(v float64) func() float64 {
	return func() float64 { return v }
}

var _ = Describe("initialMaintenanceDelay", func() {
	DescribeTable("stays within [0, min(interval, 10m))",
		func(interval time.Duration, r float64, expected time.Duration) {
			Expect(initialMaintenanceDelay(interval, constRand(r))).To(
				BeNumerically("~", expected, time.Millisecond))
		},
		Entry("zero random value with the default interval", 30*time.Minute, 0.0, time.Duration(0)),
		Entry("half random value with the default interval", 30*time.Minute, 0.5, 5*time.Minute),
		Entry("maximum random value with the default interval", 30*time.Minute, almostOne, 10*time.Minute),
		Entry("maximum random value with a long interval", 6*time.Hour, almostOne, 10*time.Minute),
		Entry("maximum random value with a short interval", 30*time.Second, almostOne, 30*time.Second),
		Entry("half random value with a short interval", 30*time.Second, 0.5, 15*time.Second),
	)

	It("is below the cap for every random value", func() {
		Expect(initialMaintenanceDelay(time.Hour, constRand(almostOne))).To(
			BeNumerically("<", 10*time.Minute))
	})
})

var _ = Describe("jitterMaintenancePeriod", func() {
	DescribeTable("adds up to min(10% of the period, 5m)",
		func(period time.Duration, r float64, expected time.Duration) {
			Expect(jitterMaintenancePeriod(period, constRand(r))).To(
				BeNumerically("~", expected, time.Millisecond))
		},
		Entry("zero random value", 30*time.Minute, 0.0, 30*time.Minute),
		Entry("maximum random value with the default interval", 30*time.Minute, almostOne, 33*time.Minute),
		Entry("half random value with the default interval", 30*time.Minute, 0.5, 31*time.Minute+30*time.Second),
		Entry("maximum random value with a long interval", 6*time.Hour, almostOne, 6*time.Hour+5*time.Minute),
		Entry("maximum random value with the retry interval", 5*time.Minute, almostOne, 5*time.Minute+30*time.Second),
	)

	It("never shortens the period", func() {
		Expect(jitterMaintenancePeriod(time.Minute, constRand(0))).To(Equal(time.Minute))
	})
})
