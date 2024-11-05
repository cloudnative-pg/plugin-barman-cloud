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
	mux           *sync.Mutex
	ttl           int64
}

// NewExtendedClient returns an extended client capable of caching secrets on the 'Get' operation
func NewExtendedClient(baseClient client.Client, ttl int64) client.Client {
	return &ExtendedClient{
		Client: baseClient,
		ttl:    ttl,
	}
}

func (e *ExtendedClient) Get(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	if e.isCacheDisabled() {
		return e.Client.Get(ctx, key, obj, opts...)
	}

	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Get(ctx, key, obj, opts...)
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	// check if in cache
	for _, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			if !e.isExpired(cache.fetchUnixTime) {
				cache.secret.DeepCopyInto(obj.(*corev1.Secret))
				return nil
			}
			break
		}
	}

	if err := e.Client.Get(ctx, key, obj); err != nil {
		return err
	}

	secret := obj.(*corev1.Secret)

	// check if the secret is already in cache if so replace it
	for _, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			cache.secret = secret.DeepCopy()
			cache.fetchUnixTime = time.Now().Unix()
			return nil
		}
	}

	// otherwise add it to the cache
	e.cachedSecrets = append(e.cachedSecrets, &cachedSecret{
		secret:        secret.DeepCopy(),
		fetchUnixTime: time.Now().Unix(),
	})

	return nil
}

func (e *ExtendedClient) isExpired(unixTime int64) bool {
	return time.Now().Unix()-unixTime > e.ttl
}

func (e *ExtendedClient) isCacheDisabled() bool {
	const noCache = 0
	return e.ttl == noCache
}

// RemoveSecret ensures that a secret is not present in the cache
func (e *ExtendedClient) RemoveSecret(key client.ObjectKey) {
	if e.isCacheDisabled() {
		return
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	for i, cache := range e.cachedSecrets {
		if cache.secret.Namespace == key.Namespace && cache.secret.Name == key.Name {
			e.cachedSecrets = append(e.cachedSecrets[:i], e.cachedSecrets[i+1:]...)
			return
		}
	}
}
