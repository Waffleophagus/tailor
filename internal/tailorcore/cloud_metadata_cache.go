package tailorcore

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

// Every open browser tab re-reads the topology every 2s, and enrichment costs one
// Cloud API call per device for posture attributes. Device metadata, users, posture
// and services change slowly, so share them across callers for a short TTL to stay
// well under Tailscale's API rate limits. Live state (online, peers) still comes
// from LocalAPI on every tick.
const (
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
	return c.devices.get(c.tailnet(), func() ([]cloudapi.Device, error) { return c.source.Devices(ctx) })
}

func (c *cachedCloudMetadata) Users(ctx context.Context) ([]cloudapi.User, error) {
	return c.users.get(c.tailnet(), func() ([]cloudapi.User, error) { return c.source.Users(ctx) })
}

func (c *cachedCloudMetadata) DevicePostureAttributes(ctx context.Context, deviceID string) (cloudapi.DevicePostureAttributes, error) {
	return c.posture.get(c.tailnet()+"|"+deviceID, func() (cloudapi.DevicePostureAttributes, error) {
		return c.source.DevicePostureAttributes(ctx, deviceID)
	})
}

func (c *cachedCloudMetadata) VIPServices(ctx context.Context) ([]cloudapi.VIPService, error) {
	return c.services.get(c.tailnet(), func() ([]cloudapi.VIPService, error) { return c.source.VIPServices(ctx) })
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
	mu      sync.Mutex
	value   T
	err     error
	expires time.Time
}

func (c *ttlCache[T]) get(key string, fetch func() (T, error)) (T, error) {
	entry := c.entry(key)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	now := time.Now()
	if now.Before(entry.expires) {
		return entry.value, entry.err
	}
	value, err := fetch()
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// The caller went away; don't make the next caller wait out an error TTL.
		var zero T
		return zero, err
	}
	ttl := cloudMetadataTTL
	if err != nil {
		ttl = cloudMetadataErrorTTL
	}
	entry.value, entry.err, entry.expires = value, err, now.Add(ttl)
	return value, err
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
				if now.After(e.expires) {
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
