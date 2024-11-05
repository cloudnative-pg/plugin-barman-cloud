package client

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cachedSecret struct {
	secret        *corev1.Secret
	fetchUnixTime int64
}

// ExtendedClient is an extended client that is capable of caching multiple secrets without relying on 'list and watch'
type ExtendedClient struct {
	client.Client
	cachedSecrets []*cachedSecret
	// add a mux to lock the operations on the cache
	mux *sync.Mutex
}

// NewExtendedClient returns an extended client capable of caching secrets on the 'Get' operation
func NewExtendedClient(baseClient client.Client) client.Client {
	return &ExtendedClient{
		Client: baseClient,
	}
}

func (e *ExtendedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Get(ctx, key, obj, opts...)
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	// check if in cache
	for _, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			if time.Now().Unix()-cache.fetchUnixTime < 180 {
				cache.secret.DeepCopyInto(obj.(*corev1.Secret))
				return nil
			}
			break
		}
	}

	if err := e.Client.Get(ctx, key, obj); err != nil {
		return err
	}

	// check if the secret is already in cache if so replace it
	for _, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			cache.secret = obj.(*corev1.Secret)
			cache.fetchUnixTime = time.Now().Unix()
			return nil
		}
	}

	if secret, ok := obj.(*corev1.Secret); ok {
		e.cachedSecrets = append(e.cachedSecrets, &cachedSecret{
			secret:        secret,
			fetchUnixTime: time.Now().Unix(),
		})
	}

	return nil
}

// RemoveSecret ensures that a secret is not present in the cache
func (e *ExtendedClient) RemoveSecret(key client.ObjectKey) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for i, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			e.cachedSecrets = append(e.cachedSecrets[:i], e.cachedSecrets[i+1:]...)
			return
		}
	}
}
