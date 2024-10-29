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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"gopkg.in/yaml.v3"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// ApplyKustomizationOptions holds options for applying kustomizations
type ApplyKustomizationOptions struct {
	IgnoreExistingResources bool
}

// ApplyKustomizationOption is a functional option for ApplyKustomization
type ApplyKustomizationOption func(*ApplyKustomizationOptions)

// ApplyKustomization builds the kustomization and creates the resources
func ApplyKustomization(
	ctx context.Context,
	cl client.Client,
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

	return applyResourceMap(ctx, cl, resourceMap)
}

func applyResourceMap(ctx context.Context, cl client.Client, resourceMap resmap.ResMap) error {
	yamlBytes, err := resourceMap.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to convert resources to YAML: %w", err)
	}
	r := bytes.NewReader(yamlBytes)
	dec := yaml.NewDecoder(r)
	for {
		// parse the YAML doc
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		err := dec.Decode(obj.Object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("could not decode object: %w", err)
		}
		if obj.Object == nil {
			continue
		}
		if err := applyResource(ctx, cl, obj); err != nil {
			return err
		}
	}

	return nil
}

func applyResource(ctx context.Context, cl client.Client, obj *unstructured.Unstructured) error {
	if err := cl.Create(ctx, obj); err != nil {
		if apimachineryerrors.IsAlreadyExists(err) {
			// If the resource already exists, retrieve the existing resource
			existing := &unstructured.Unstructured{}
			existing.SetGroupVersionKind(obj.GroupVersionKind())
			key := client.ObjectKey{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			}
			if err := cl.Get(ctx, key, existing); err != nil {
				log.Fatalf("Error getting existing resource: %v", err)
			}

			// Update the existing resource with the new data
			obj.SetResourceVersion(existing.GetResourceVersion())
			err = cl.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("error updating resource: %v", err)
			}
		} else {
			return fmt.Errorf("error creating resource: %v", err)
		}
	}
	return nil
}
