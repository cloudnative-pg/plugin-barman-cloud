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
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/certmanager"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cloudnativepg"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/kind"
)

// SetupOptions contains the options for setting up the test environment.
type SetupOptions struct {
	K8sVersion string

	KindVersion            string
	KindClusterNamePrefix  string
	KindAdditionalNetworks []string

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

// WithK8sVersion sets the Kubernetes version for the test environment.
func WithK8sVersion(version string) SetupOption {
	return func(opts *SetupOptions) {
		opts.K8sVersion = version
	}
}

// WithKindVersion sets the Kind version for the test environment.
func WithKindVersion(version string) SetupOption {
	return func(opts *SetupOptions) {
		opts.KindVersion = version
	}
}

// WithKindAdditionalNetworks sets the additional networks for the Kind cluster for the test environment.
func WithKindAdditionalNetworks(networks []string) SetupOption {
	return func(opts *SetupOptions) {
		opts.KindAdditionalNetworks = networks
	}
}

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

// WithKindClusterNamePrefix sets the prefix for the Kind cluster name for the test environment.
func withKindClusterNamePrefix(name string) SetupOption {
	return func(opts *SetupOptions) {
		opts.KindClusterNamePrefix = name
	}
}

const (
	kindConfigFile = "config/kind-config.yaml"
)

func defaultSetupOptions() SetupOptions {
	// TODO: renovate
	return SetupOptions{
		K8sVersion:             "v1.31.1",
		KindVersion:            "v0.24.0",
		CertManagerVersion:     "v1.15.1",
		KindClusterNamePrefix:  "e2e",
		KindAdditionalNetworks: []string{},
	}
}

// Setup sets up the test environment for the e2e tests, starting kind and installing the necessary components.
func Setup(ctx context.Context, opts ...SetupOption) (client.Client, error) {
	options := defaultSetupOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if err := setupKind(ctx, options); err != nil {
		return nil, err
	}

	cl, err := getClient()
	if err != nil {
		return nil, err
	}

	if err := installCertManager(ctx, cl, options); err != nil {
		return nil, err
	}

	if err := installCNPG(ctx, cl, options); err != nil {
		return nil, err
	}

	// Return the Kubernetes client used for the tests
	return cl, nil
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

func getClient() (client.Client, error) {
	// Use the current kubernetes client configuration
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return cl, nil
}

func setupKind(ctx context.Context, options SetupOptions) error {
	// This function sets up the environment for the e2e tests
	// by creating the cluster and installing the necessary
	// components.
	expectedClusterName := kindClusterName(options.KindClusterNamePrefix, options.K8sVersion)
	provider := cluster.NewProvider()
	clusterIsRunning, err := kind.IsClusterRunning(provider, expectedClusterName)
	if err != nil {
		return fmt.Errorf("failed to check if Kind cluster is running: %w", err)
	}
	if !clusterIsRunning {
		kindOpts := []kind.CreateClusterOption{
			kind.WithK8sVersion(options.K8sVersion),
			kind.WithConfigFile(kindConfigFile),
			kind.WithNetworks(options.KindAdditionalNetworks),
		}
		if err := kind.CreateCluster(ctx, provider, expectedClusterName, kindOpts...); err != nil {
			return fmt.Errorf("failed to create Kind cluster: %w", err)
		}
	}

	return nil
}

func kindClusterName(prefix, k8sVersion string) string {
	k8sVersion = strings.ReplaceAll(k8sVersion, ".", "-")
	return fmt.Sprintf("%s-%s", prefix, k8sVersion)
}
