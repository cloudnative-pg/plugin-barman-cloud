package client

import (
	"context"
	"sync"
	"time"

	"github.com/cloudnative-pg/machinery/pkg/log"
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
		mux:    &sync.Mutex{},
	}
}

func (e *ExtendedClient) Get(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	contextLogger := log.FromContext(ctx).
		WithName("extended_client").
		WithValues("name", key.Name, "namespace", key.Namespace)

	if e.isCacheDisabled() {
		return e.Client.Get(ctx, key, obj, opts...)
	}

	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Get(ctx, key, obj, opts...)
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	expiredSecretIndex := -1
	// check if in cache
	for idx, cache := range e.cachedSecrets {
		if cache.secret.Namespace != key.Namespace || cache.secret.Name != key.Name {
			continue
		}
		if e.isExpired(cache.fetchUnixTime) {
			contextLogger.Trace("secret found, but it is expired")
			expiredSecretIndex = idx
			break
		}
		contextLogger.Trace("secret found, loading it from cache")
		cache.secret.DeepCopyInto(obj.(*corev1.Secret))
		return nil
	}

	if err := e.Client.Get(ctx, key, obj); err != nil {
		return err
	}

	cs := &cachedSecret{
		secret:        obj.(*corev1.Secret).DeepCopy(),
		fetchUnixTime: time.Now().Unix(),
	}

	if expiredSecretIndex != -1 {
		e.cachedSecrets[expiredSecretIndex] = cs
	} else {
		e.cachedSecrets = append(e.cachedSecrets, cs)
	}

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

func (e *ExtendedClient) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.UpdateOption,
) error {
	if e.isCacheDisabled() {
		return e.Client.Update(ctx, obj, opts...)
	}

	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Update(ctx, obj, opts...)
	}

	e.RemoveSecret(client.ObjectKeyFromObject(obj))

	return e.Client.Update(ctx, obj, opts...)
}

func (e *ExtendedClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	if e.isCacheDisabled() {
		return e.Client.Delete(ctx, obj, opts...)
	}

	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Delete(ctx, obj, opts...)
	}

	e.RemoveSecret(client.ObjectKeyFromObject(obj))

	return e.Client.Delete(ctx, obj, opts...)
}

func (e *ExtendedClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	if e.isCacheDisabled() {
		return e.Client.Patch(ctx, obj, patch, opts...)
	}

	if _, ok := obj.(*corev1.Secret); !ok {
		return e.Client.Patch(ctx, obj, patch, opts...)
	}

	e.RemoveSecret(client.ObjectKeyFromObject(obj))

	return e.Client.Patch(ctx, obj, patch, opts...)
}
