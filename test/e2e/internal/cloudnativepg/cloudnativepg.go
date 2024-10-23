/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudnativepg

import (
	"context"
	"fmt"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types2 "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/resid"

	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/deployment"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/kustomize"
)

// InstallCloudNativePGOptions contains the options for installing CloudNativePG
type InstallCloudNativePGOptions struct {
	ImageName                string
	ImageTag                 string
	KustomizationResourceURL string
	KustomizationRef         string
	KustomizationTimeout     string
	IgnoreExistResources     bool
}

// InstallOption is a function that sets up an option for installing CloudNativePG
type InstallOption func(*InstallCloudNativePGOptions)

// WithImageName sets the name for the CloudNativePG image
func WithImageName(ref string) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.ImageName = ref
	}
}

// WithImageTag sets the tag for the CloudNativePG image
func WithImageTag(tag string) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.ImageTag = tag
	}
}

// WithKustomizationResourceURL sets the URL for the CloudNativePG kustomization
func WithKustomizationResourceURL(url string) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.KustomizationResourceURL = url
	}
}

// WithKustomizationRef sets the ref for the CloudNativePG kustomization
func WithKustomizationRef(ref string) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.KustomizationRef = ref
	}
}

// WithKustomizationTimeout sets the timeout for the kustomization resources
func WithKustomizationTimeout(timeout string) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.KustomizationTimeout = timeout
	}
}

// WithIgnoreExistingResources sets whether to ignore existing resources
func WithIgnoreExistingResources(ignore bool) InstallOption {
	return func(opts *InstallCloudNativePGOptions) {
		opts.IgnoreExistResources = ignore
	}
}

// Install installs CloudNativePG using kubectl
func Install(ctx context.Context, cl client.Client, opts ...InstallOption) error {
	// Defining the default options
	options := &InstallCloudNativePGOptions{
		ImageName:                "ghcr.io/cloudnative-pg/cloudnative-pg-testing",
		ImageTag:                 "main",
		KustomizationResourceURL: "https://github.com/cloudnative-pg/cloudnative-pg.git/config/default",
		KustomizationRef:         "main",
		KustomizationTimeout:     "120",
		IgnoreExistResources:     true,
	}

	for _, opt := range opts {
		opt(options)
	}
	kustomizationFullURL := fmt.Sprintf("%s/?ref=%s&timeout=%s", options.KustomizationResourceURL,
		options.KustomizationRef, options.KustomizationTimeout)

	// Generate the Kustomization
	kustomization := &types.Kustomization{
		Resources: []string{kustomizationFullURL},
		Images: []types.Image{
			{
				Name:    "controller",
				NewName: options.ImageName,
				NewTag:  options.ImageTag,
			},
		},
		Patches: []types.Patch{
			{
				Patch: fmt.Sprintf(`[{"op": "replace", "path": "/spec/template/spec/containers/0/env/0/value", "value": "%v:%v"}]`,
					options.ImageName, options.ImageTag),
				Target: &types.Selector{
					ResId: resid.ResId{
						Gvk:       resid.Gvk{Kind: "Deployment", Version: "v1", Group: "apps"},
						Name:      "cnpg-controller-manager",
						Namespace: "cnpg-system",
					},
				},
				Options: nil,
			},
		},
	}

	scheme, err := setupScheme()
	if err != nil {
		return err
	}

	if err := kustomize.ApplyKustomization(ctx, cl, scheme, kustomization,
		kustomize.WithIgnoreExistingResources(options.IgnoreExistResources)); err != nil {
		return fmt.Errorf("failed to apply kustomization: %w", err)
	}

	// Set default timeout if none is provided
	const defaultTimeout = 5 * time.Minute

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	if err := deployment.WaitForDeploymentReady(ctx, cl, types2.NamespacedName{
		Namespace: "cnpg-system",
		Name:      "cnpg-controller-manager",
	}, 5*time.Second); err != nil {
		return fmt.Errorf("failed to wait for deployment to be ready: %w", err)
	}

	return nil
}

func setupScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add core/v1 to scheme: %w", err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add apiextensions to scheme: %w", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add apps/v1 to scheme: %w", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add rbac/v1 to scheme: %w", err)
	}
	if err := admissionregistrationv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add admissionregistration/v1 to scheme: %w", err)
	}
	return scheme, nil
}
