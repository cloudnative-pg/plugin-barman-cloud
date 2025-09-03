package client

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/cloudnative-pg/machinery/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pluginBarman "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// DefaultTTLSeconds is the default TTL in seconds of cache entries
const DefaultTTLSeconds = 10

type cachedEntry struct {
	entry         client.Object
	fetchUnixTime int64
	ttlSeconds    int64
}

func (e *cachedEntry) isExpired() bool {
	return time.Now().Unix()-e.fetchUnixTime > e.ttlSeconds
}

// ExtendedClient is an extended client that is capable of caching multiple secrets without relying on informers
type ExtendedClient struct {
	client.Client
	cachedObjects []cachedEntry
	mux           *sync.Mutex
}

// NewExtendedClient returns an extended client capable of caching secrets on the 'Get' operation
func NewExtendedClient(
	baseClient client.Client,
) client.Client {
	return &ExtendedClient{
		Client: baseClient,
		mux:    &sync.Mutex{},
	}
}

func (e *ExtendedClient) isObjectCached(obj client.Object) bool {
	if _, isSecret := obj.(*corev1.Secret); isSecret {
		return true
	}

	if _, isObjectStore := obj.(*pluginBarman.ObjectStore); isObjectStore {
		return true
	}

	return false
}

// Get behaves like the original Get method, but uses a cache for secrets
func (e *ExtendedClient) Get(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	if e.isObjectCached(obj) {
		return e.getCachedObject(ctx, key, obj, opts...)
	}

	return e.Client.Get(ctx, key, obj, opts...)
}

func (e *ExtendedClient) getCachedObject(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	contextLogger := log.FromContext(ctx).
		WithName("extended_client").
		WithValues("name", key.Name, "namespace", key.Namespace)

	contextLogger.Trace("locking the cache")
	e.mux.Lock()
	defer e.mux.Unlock()

	// check if in cache
	expiredObjectIndex := -1
	for idx, cacheEntry := range e.cachedObjects {
		if cacheEntry.entry.GetNamespace() != key.Namespace || cacheEntry.entry.GetName() != key.Name {
			continue
		}
		if reflect.TypeOf(cacheEntry.entry) != reflect.TypeOf(obj) {
			continue
		}
		if cacheEntry.isExpired() {
			contextLogger.Trace("expired object found")
			expiredObjectIndex = idx
			break
		}

		contextLogger.Debug("object found, loading it from cache")

		// Yes, this is a terrible hack, but that's exactly the way
		// controller-runtime works.
		// https://github.com/kubernetes-sigs/controller-runtime/blob/
		// 717b32aede14c921d239cf1b974a11e718949865/pkg/cache/internal/cache_reader.go#L92
		outVal := reflect.ValueOf(obj)
		objVal := reflect.ValueOf(cacheEntry.entry)
		if !objVal.Type().AssignableTo(outVal.Type()) {
			return fmt.Errorf("cache had type %s, but %s was asked for", objVal.Type(), outVal.Type())
		}

		reflect.Indirect(outVal).Set(reflect.Indirect(objVal))
		return nil
	}

	if err := e.Client.Get(ctx, key, obj, opts...); err != nil {
		return err
	}

	cs := cachedEntry{
		entry:         obj.DeepCopyObject().(client.Object),
		fetchUnixTime: time.Now().Unix(),
		ttlSeconds:    DefaultTTLSeconds,
	}

	contextLogger.Debug("setting object in the cache")
	if expiredObjectIndex != -1 {
		e.cachedObjects[expiredObjectIndex] = cs
	} else {
		e.cachedObjects = append(e.cachedObjects, cs)
	}

	return nil
}

// removeObject ensures that a object is not present in the cache
func (e *ExtendedClient) removeObject(object client.Object) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for i, cache := range e.cachedObjects {
		if cache.entry.GetNamespace() == object.GetNamespace() &&
			cache.entry.GetName() == object.GetName() &&
			reflect.TypeOf(cache.entry) == reflect.TypeOf(object) {
			e.cachedObjects = append(e.cachedObjects[:i], e.cachedObjects[i+1:]...)
			return
		}
	}
}

// Update behaves like the original Update method, but on secrets it removes the secret from the cache
func (e *ExtendedClient) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.UpdateOption,
) error {
	if e.isObjectCached(obj) {
		e.removeObject(obj)
	}

	return e.Client.Update(ctx, obj, opts...)
}

// Delete behaves like the original Delete method, but on secrets it removes the secret from the cache
func (e *ExtendedClient) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	if e.isObjectCached(obj) {
		e.removeObject(obj)
	}

	return e.Client.Delete(ctx, obj, opts...)
}

// Patch behaves like the original Patch method, but on secrets it removes the secret from the cache
func (e *ExtendedClient) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.PatchOption,
) error {
	if e.isObjectCached(obj) {
		e.removeObject(obj)
	}

	return e.Client.Patch(ctx, obj, patch, opts...)
}
