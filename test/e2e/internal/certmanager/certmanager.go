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

package certmanager

import (
	"context"
	"fmt"
	"time"

	types2 "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/deployment"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/kustomize"
)

// InstallOptions contains the options for installing cert-manager.
type InstallOptions struct {
	Version              string
	IgnoreExistResources bool
}

// InstallOption is a function that sets up an option for installing cert-manager.
type InstallOption func(*InstallOptions)

// WithVersion sets the version of cert-manager to install.
func WithVersion(version string) InstallOption {
	return func(opts *InstallOptions) {
		opts.Version = version
	}
}

// WithIgnoreExistingResources sets whether to ignore existing resources.
func WithIgnoreExistingResources(ignore bool) InstallOption {
	return func(opts *InstallOptions) {
		opts.IgnoreExistResources = ignore
	}
}

// TODO: renovate

// DefaultVersion is the default version of cert-manager to install.
const DefaultVersion = "v1.15.1"

// Install installs cert-manager using kubectl.
func Install(ctx context.Context, cl client.Client, opts ...InstallOption) error {
	options := &InstallOptions{
		Version:              DefaultVersion,
		IgnoreExistResources: true,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Define the KustomizationResourceURL for the cert-manager manifests
	url := fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml",
		options.Version)

	// Generate the Kustomization
	kustomization := &types.Kustomization{
		Resources: []string{url},
	}

	// Add all the resources defined in the cert-manager manifests
	if err := kustomize.ApplyKustomization(ctx, cl, kustomization); err != nil {
		return fmt.Errorf("failed to apply kustomization: %w", err)
	}

	// Set default timeout if none is provided
	const defaultTimeout = 5 * time.Minute

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	deployments := []string{"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"}
	interval := 5 * time.Second
	for _, deploymentName := range deployments {
		if err := deployment.WaitForDeploymentReady(ctx, cl, types2.NamespacedName{
			Namespace: "cert-manager",
			Name:      deploymentName,
		}, interval); err != nil {
			return fmt.Errorf("failed to wait for deployment %s to be ready: %w", deploymentName, err)
		}
	}

	return nil
}
