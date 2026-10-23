package tailorcore

import (
	"context"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

// Every open browser tab re-reads the topology every 2s, and enrichment costs one
// Cloud API call per device for posture attributes. Device metadata, users, posture
// and services change slowly, so share them across callers for a short TTL to stay
// well under Tailscale's API rate limits. Live state (online, peers) still comes
// from LocalAPI on every tick.
// Variables so tests can shorten them.
var (
	cloudMetadataTTL      = 30 * time.Second
	cloudMetadataErrorTTL = 5 * time.Second
)

// cloudMetadata is the read-only Cloud API surface used to enrich topology.
type cloudMetadata interface {
	Devices(context.Context) ([]cloudapi.Device, error)
	Users(context.Context) ([]cloudapi.User, error)
	DevicePostureAttributes(context.Context, string) (cloudapi.DevicePostureAttributes, error)
	VIPServices(context.Context) ([]cloudapi.VIPService, error)
}

// cachedCloudMetadata memoizes cloudMetadata per tailnet. Callers must treat
// returned values as read-only because they are shared.
type cachedCloudMetadata struct {
	source  cloudMetadata
	tailnet func() string

	devices  ttlCache[[]cloudapi.Device]
	users    ttlCache[[]cloudapi.User]
	posture  ttlCache[cloudapi.DevicePostureAttributes]
	services ttlCache[[]cloudapi.VIPService]
}

func newCachedCloudMetadata(source cloudMetadata, tailnet func() string) *cachedCloudMetadata {
	return &cachedCloudMetadata{source: source, tailnet: tailnet}
}

func (c *cachedCloudMetadata) Devices(ctx context.Context) ([]cloudapi.Device, error) {
	return c.devices.get(ctx, c.tailnet(), func() ([]cloudapi.Device, error) { return c.source.Devices(ctx) })
}

func (c *cachedCloudMetadata) Users(ctx context.Context) ([]cloudapi.User, error) {
	return c.users.get(ctx, c.tailnet(), func() ([]cloudapi.User, error) { return c.source.Users(ctx) })
}

func (c *cachedCloudMetadata) DevicePostureAttributes(ctx context.Context, deviceID string) (cloudapi.DevicePostureAttributes, error) {
	return c.posture.get(ctx, c.tailnet()+"|"+deviceID, func() (cloudapi.DevicePostureAttributes, error) {
		return c.source.DevicePostureAttributes(ctx, deviceID)
	})
}

func (c *cachedCloudMetadata) VIPServices(ctx context.Context) ([]cloudapi.VIPService, error) {
	return c.services.get(ctx, c.tailnet(), func() ([]cloudapi.VIPService, error) { return c.source.VIPServices(ctx) })
}

// Invalidate drops all cached metadata, e.g. after re-authentication.
func (c *cachedCloudMetadata) Invalidate() {
	c.devices.clear()
	c.users.clear()
	c.posture.clear()
	c.services.clear()
}

// ttlCache caches one value per key. Concurrent callers for the same key wait for
// a single in-flight fetch instead of each calling the API.
type ttlCache[T any] struct {
	mu        sync.Mutex
	entries   map[string]*ttlEntry[T]
	lastPurge time.Time
}

type ttlEntry[T any] struct {
	mu       sync.Mutex // held briefly, never across a fetch
	value    T
	err      error
	expires  time.Time
	inflight chan struct{} // non-nil while one caller fetches; closed when done
}

// get returns the cached value for key, or runs fetch. Concurrent callers share a
// single in-flight fetch, but each stops waiting as soon as its own ctx ends.
func (c *ttlCache[T]) get(ctx context.Context, key string, fetch func() (T, error)) (T, error) {
	for {
		entry := c.entry(key)
		entry.mu.Lock()
		if time.Now().Before(entry.expires) {
			value, err := entry.value, entry.err
			entry.mu.Unlock()
			return value, err
		}
		if wait := entry.inflight; wait != nil {
			entry.mu.Unlock()
			select {
			case <-wait:
				continue // re-check: the fetch may have been cancelled and stored nothing
			case <-ctx.Done():
				var zero T
				return zero, ctx.Err()
			}
		}
		done := make(chan struct{})
		entry.inflight = done
		entry.mu.Unlock()

		value, err := fetch()

		entry.mu.Lock()
		entry.inflight = nil
		// A caller that went away stores nothing, so the next caller fetches
		// afresh. Ask its context: an upstream HTTP timeout also looks like
		// context.DeadlineExceeded but must be cached like any other error.
		if ctx.Err() == nil {
			ttl := cloudMetadataTTL
			if err != nil {
				ttl = cloudMetadataErrorTTL
			}
			// The TTL starts when the data arrives, not when the fetch began.
			entry.value, entry.err, entry.expires = value, err, time.Now().Add(ttl)
		}
		entry.mu.Unlock()
		close(done)
		return value, err
	}
}

func (c *ttlCache[T]) entry(key string) *ttlEntry[T] {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]*ttlEntry[T]{}
	}
	now := time.Now()
	// Per-device keys churn with ephemeral nodes; drop stale ones periodically.
	if now.Sub(c.lastPurge) > cloudMetadataTTL {
		for k, e := range c.entries {
			if e.mu.TryLock() {
				if e.inflight == nil && now.After(e.expires) {
					delete(c.entries, k)
				}
				e.mu.Unlock()
			}
		}
		c.lastPurge = now
	}
	entry := c.entries[key]
	if entry == nil {
		entry = &ttlEntry[T]{}
		c.entries[key] = entry
	}
	return entry
}

func (c *ttlCache[T]) clear() {
	c.mu.Lock()
	c.entries = nil
	c.mu.Unlock()
}
