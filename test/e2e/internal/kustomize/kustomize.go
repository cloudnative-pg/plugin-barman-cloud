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

package kustomize

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// ApplyKustomizationOptions holds options for applying kustomizations
type ApplyKustomizationOptions struct {
	IgnoreExistingResources bool
}

// ApplyKustomizationOption is a functional option for ApplyKustomization
type ApplyKustomizationOption func(*ApplyKustomizationOptions)

// WithIgnoreExistingResources sets the ignore existing option
func WithIgnoreExistingResources(ignore bool) ApplyKustomizationOption {
	return func(opts *ApplyKustomizationOptions) {
		opts.IgnoreExistingResources = ignore
	}
}

// ApplyKustomization builds the kustomization and creates the resources
func ApplyKustomization(
	ctx context.Context,
	cl client.Client,
	scheme *runtime.Scheme,
	kustomization *types.Kustomization,
	options ...ApplyKustomizationOption,
) error {
	opts := &ApplyKustomizationOptions{
		IgnoreExistingResources: true,
	}
	for _, opt := range options {
		opt(opts)
	}

	// We'd rather use an in-memory filesystem, but krusty doesn't support it yet for git URLs
	// See https://github.com/kubernetes-sigs/kustomize/issues/4390
	// Create an in-memory filesystem
	fSys := filesys.MakeFsOnDisk()

	// Write the Kustomization to the filesystem
	kustomizationYAML, err := yaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization: %w", err)
	}

	err = fSys.WriteFile("kustomization.yaml", kustomizationYAML)
	if err != nil {
		return fmt.Errorf("failed to write kustomization.yaml: %w", err)
	}
	defer fSys.RemoveAll("kustomization.yaml") //nolint:errcheck

	// Build the Kustomization
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resourceMap, err := k.Run(fSys, ".")
	if err != nil {
		return fmt.Errorf("failed to run kustomize: %w", err)
	}

	codecs := serializer.NewCodecFactory(scheme)
	deserializer := codecs.UniversalDeserializer()

	// Apply the resources
	for _, res := range resourceMap.Resources() {
		resJSON, err := res.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to convert resource map to yaml: %w", err)
		}

		obj, _, err := deserializer.Decode(resJSON, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to decode resource: %w", err)
		}
		// TODO: review
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return fmt.Errorf("failed to convert object to unstructured: %w", err)
		}
		u := &unstructured.Unstructured{Object: unstructuredObj}

		if err := cl.Create(ctx, u); err != nil {
			if errors.IsAlreadyExists(err) && opts.IgnoreExistingResources {
				continue
			}
			return fmt.Errorf("failed to apply resource: %w", err)
		}
	}

	return nil
}
