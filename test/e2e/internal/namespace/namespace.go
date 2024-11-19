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

package namespace

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateUniqueNamespace creates a namespace with an unique suffix.
func CreateUniqueNamespace(ctx context.Context, cl client.Client, prefix string) (*corev1.Namespace, error) {
	for {
		randInt, err := rand.Int(rand.Reader, big.NewInt(100000))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random number: %w", err)
		}
		namespaceName := fmt.Sprintf("%s-%d", prefix, randInt)
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		err = cl.Create(ctx, namespace)
		if err == nil {
			return namespace, nil
		}
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create namespace: %w", err)
		}
	}
}
