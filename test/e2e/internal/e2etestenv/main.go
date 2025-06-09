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

package e2etestenv

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/certmanager"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cloudnativepg"
)

// SetupOptions contains the options for setting up the test environment.
type SetupOptions struct {
	CNPGKustomizationURL     string
	CNPGKustomizationRef     string
	CNPGKustomizationTimeout string
	CNPGImageName            string
	CNPGImageTag             string

	CertManagerVersion string

	IgnoreExistingResources bool
}

// SetupOption is a function that sets up an option for the test environment setup.
type SetupOption func(*SetupOptions)

// WithCNPGKustomizationURL sets the CloudNativePG kustomization URL for the test environment.
func WithCNPGKustomizationURL(url string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CNPGKustomizationURL = url
	}
}

// WithCNPGKustomizationRef sets the CloudNativePG kustomization ref for the test environment.
func WithCNPGKustomizationRef(ref string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CNPGKustomizationRef = ref
	}
}

// WithCNPGKustomizationTimeout sets the CloudNativePG kustomization timeout for the test environment.
func WithCNPGKustomizationTimeout(timeout string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CNPGKustomizationTimeout = timeout
	}
}

// WithCNPGImageName sets the CloudNativePG image name for the test environment.
func WithCNPGImageName(name string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CNPGImageName = name
	}
}

// WithCNPGImageTag sets the CloudNativePG image tag for the test environment.
func WithCNPGImageTag(tag string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CNPGImageTag = tag
	}
}

// WithCertManagerVersion sets the cert-manager version for the test environment.
func WithCertManagerVersion(version string) SetupOption {
	return func(opts *SetupOptions) {
		opts.CertManagerVersion = version
	}
}

// WithIgnoreExistingResources sets the option to ignore existing resources when creating the test environment,
// instead of returning an error.
func WithIgnoreExistingResources(ignore bool) SetupOption {
	return func(opts *SetupOptions) {
		opts.IgnoreExistingResources = ignore
	}
}

func defaultSetupOptions() SetupOptions {
	// TODO: renovate
	return SetupOptions{
		CertManagerVersion: "v1.15.1",
	}
}

// Setup sets up the test environment for the e2e tests, starting kind and installing the necessary components.
//
//nolint:ireturn
func Setup(ctx context.Context, cl client.Client, opts ...SetupOption) error {
	options := defaultSetupOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if err := installCertManager(ctx, cl, options); err != nil {
		return err
	}

	options.CNPGImageTag = "dev-config-plugin"
	if err := installCNPG(ctx, cl, options); err != nil {
		return err
	}

	return nil
}

func installCNPG(ctx context.Context, cl client.Client, options SetupOptions) error {
	// Install CloudNativePG
	var cnpgIstallOptions []cloudnativepg.InstallOption
	if options.CNPGKustomizationURL != "" {
		cnpgIstallOptions = append(cnpgIstallOptions,
			cloudnativepg.WithKustomizationResourceURL(options.CNPGKustomizationURL))
	}
	if options.CNPGKustomizationRef != "" {
		cnpgIstallOptions = append(cnpgIstallOptions, cloudnativepg.WithKustomizationRef(options.CNPGKustomizationRef))
	}
	if options.CNPGKustomizationTimeout != "" {
		cnpgIstallOptions = append(cnpgIstallOptions,
			cloudnativepg.WithKustomizationTimeout(options.CNPGKustomizationTimeout))
	}
	if options.CNPGImageName != "" {
		cnpgIstallOptions = append(cnpgIstallOptions, cloudnativepg.WithImageName(options.CNPGImageName))
	}
	if options.CNPGImageTag != "" {
		cnpgIstallOptions = append(cnpgIstallOptions, cloudnativepg.WithImageTag(options.CNPGImageTag))
	}
	if options.IgnoreExistingResources {
		cnpgIstallOptions = append(cnpgIstallOptions,
			cloudnativepg.WithIgnoreExistingResources(options.IgnoreExistingResources))
	}
	if err := cloudnativepg.Install(ctx, cl, cnpgIstallOptions...); err != nil {
		return fmt.Errorf("failed to install cloudnative-pg: %w", err)
	}

	return nil
}

func installCertManager(ctx context.Context, cl client.Client, options SetupOptions) error {
	// Install cert-manager
	var certManagerInstallOptions []certmanager.InstallOption
	if options.CertManagerVersion != "" {
		certManagerInstallOptions = append(certManagerInstallOptions,
			certmanager.WithVersion(options.CertManagerVersion))
	}
	if options.IgnoreExistingResources {
		certManagerInstallOptions = append(certManagerInstallOptions,
			certmanager.WithIgnoreExistingResources(options.IgnoreExistingResources))
	}
	cmCtx, cmCtxCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cmCtxCancel()
	if err := certmanager.Install(cmCtx, cl,
		certManagerInstallOptions...); err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}

	return nil
}
