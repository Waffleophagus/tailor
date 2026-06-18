package policy

import (
	"crypto/sha256"
	"encoding/json"
	"sync"

	"github.com/Waffleophagus/tailor/internal/api"
)

// Cache memoizes policy-derived values for the current raw policy. Edge results
// additionally include devices and options because topology can change without
// the policy changing.
type Cache struct {
	mu sync.Mutex

	policyHash [sha256.Size]byte
	hasPolicy  bool
	parsed     Policy
	parseErr   error
	mapValue   api.PolicyMapResponse
	mapErr     error
	edges      map[[sha256.Size]byte]edgeCacheEntry
}

type edgeCacheEntry struct {
	value []api.Edge
	err   error
}

func (c *Cache) Invalidate() {
	c.mu.Lock()
	c.hasPolicy = false
	c.parsed = Policy{}
	c.parseErr = nil
	c.mapValue = api.PolicyMapResponse{}
	c.mapErr = nil
	c.edges = nil
	c.mu.Unlock()
}

func (c *Cache) Parse(raw string) (Policy, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.selectPolicyLocked(raw)
	return c.parsed, c.parseErr
}

func (c *Cache) StructuredMap(raw string) (api.PolicyMapResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.selectPolicyLocked(raw)
	return c.mapValue, c.mapErr
}

func (c *Cache) EffectiveAccessEdges(raw string, devices []api.Device, options EdgeOptions) ([]api.Edge, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.selectPolicyLocked(raw)
	if c.parseErr != nil {
		return nil, c.parseErr
	}

	key := edgeKey(devices, options)
	if cached, ok := c.edges[key]; ok {
		return cloneEdges(cached.value), cached.err
	}
	value := ResolveEffectiveAccess(c.parsed, devices, options)
	c.edges[key] = edgeCacheEntry{value: cloneEdges(value)}
	return cloneEdges(value), nil
}

func cloneEdges(edges []api.Edge) []api.Edge {
	if len(edges) == 0 {
		return nil
	}
	out := make([]api.Edge, len(edges))
	copy(out, edges)
	return out
}

func (c *Cache) selectPolicyLocked(raw string) {
	hash := sha256.Sum256([]byte(raw))
	if c.hasPolicy && c.policyHash == hash {
		return
	}
	parsed, parseErr := Parse(raw)
	mapValue, mapErr := StructuredMap(raw)
	c.policyHash = hash
	c.hasPolicy = true
	c.parsed = parsed
	c.parseErr = parseErr
	c.mapValue = mapValue
	c.mapErr = mapErr
	c.edges = make(map[[sha256.Size]byte]edgeCacheEntry)
}

func edgeKey(devices []api.Device, options EdgeOptions) [sha256.Size]byte {
	payload, _ := json.Marshal(struct {
		Devices []api.Device
		Options EdgeOptions
	}{devices, options})
	return sha256.Sum256(payload)
}
