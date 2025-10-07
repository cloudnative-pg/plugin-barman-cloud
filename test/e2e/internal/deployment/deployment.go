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

package deployment

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsReady checks if the deployment is ready.
func IsReady(ctx context.Context, cl client.Client, name types.NamespacedName) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := cl.Get(ctx, name, deployment)
	if err != nil {
		return false, fmt.Errorf("failed to get %s deployment: %w", name, err)
	}

	// Check if the deployment is ready
	ready := false
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == "True" {
			ready = true
			break
		}
	}
	if !ready {
		return false, nil
	}

	return true, nil
}

// WaitForDeploymentReady waits for the deployment to be ready. ctx should have a timeout set.
func WaitForDeploymentReady(
	ctx context.Context, cl client.Client, namespacedName types.NamespacedName, interval time.Duration,
) error {
	err := wait.PollUntilContextCancel(ctx, interval, false,
		func(ctx context.Context) (bool, error) {
			ready, err := IsReady(ctx, cl, namespacedName)
			if err != nil {
				return false, fmt.Errorf("failed to check if %s is ready: %w", namespacedName, err)
			}
			if ready {
				return true, nil
			}

			return false, nil
		})
	if err != nil {
		return fmt.Errorf("failed to wait for %s to be ready: %w", namespacedName, err)
	}

	return nil
}
