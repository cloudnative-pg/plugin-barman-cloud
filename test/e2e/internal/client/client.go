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

package client

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	pluginBarmanCloudV1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// NewClient creates a new controller-runtime Kubernetes client.
//
//nolint:ireturn
func NewClient() (client.Client, *rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	if err := addScheme(cl); err != nil {
		return nil, nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	return cl, cfg, nil
}

// NewClientSet creates a new k8s client-go clientset.
func NewClientSet() (*kubernetes.Clientset, *rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientSet, cfg, nil
}

func addScheme(cl client.Client) error {
	scheme := cl.Scheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add core/v1 to scheme: %w", err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add apiextensions/v1 to scheme: %w", err)
	}
	if err := admissionregistrationv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add admissionregistration/v1 to scheme: %w", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add rbac/v1 to scheme: %w", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add apps/v1 to scheme: %w", err)
	}
	if err := certmanagerv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add cert-manager/v1 to scheme: %w", err)
	}
	if err := pluginBarmanCloudV1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add plugin-barman-cloud/v1 to scheme: %w", err)
	}
	if err := cloudnativepgv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add cloudnativepg/v1 to scheme: %w", err)
	}

	return nil
}
