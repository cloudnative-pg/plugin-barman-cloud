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

package objectstore

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultSize is the default size of the PVCs for the object stores.
	DefaultSize = "1Gi"
)

// Resources represents the resources required to create an object store.
type Resources struct {
	Deployment *appsv1.Deployment
	Service    *corev1.Service
	Secret     *corev1.Secret
	PVC        *corev1.PersistentVolumeClaim
}

// Create creates the object store resources.
func (osr Resources) Create(ctx context.Context, cl client.Client) error {
	if osr.PVC != nil {
		if err := cl.Create(ctx, osr.PVC); err != nil {
			return fmt.Errorf("failed to create PVC: %w", err)
		}
	}
	if osr.Secret != nil {
		if err := cl.Create(ctx, osr.Secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
	}
	if osr.Deployment != nil {
		if err := cl.Create(ctx, osr.Deployment); err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}
	if osr.Service != nil {
		if err := cl.Create(ctx, osr.Service); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	return nil
}
