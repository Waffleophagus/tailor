package tailorcore

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

type countingCloudMetadata struct {
	deviceCalls  atomic.Int32
	postureCalls atomic.Int32
	release      chan struct{}
	deviceErr    error
}

func (c *countingCloudMetadata) Devices(context.Context) ([]cloudapi.Device, error) {
	c.deviceCalls.Add(1)
	if c.release != nil {
		<-c.release
	}
	return []cloudapi.Device{{ID: "n1"}}, c.deviceErr
}

func (c *countingCloudMetadata) Users(context.Context) ([]cloudapi.User, error) { return nil, nil }

func (c *countingCloudMetadata) DevicePostureAttributes(ctx context.Context, _ string) (cloudapi.DevicePostureAttributes, error) {
	c.postureCalls.Add(1)
	return cloudapi.DevicePostureAttributes{}, ctx.Err()
}

func (c *countingCloudMetadata) VIPServices(context.Context) ([]cloudapi.VIPService, error) {
	return nil, nil
}

func newTestCloudMetadata(source cloudMetadata) *cachedCloudMetadata {
	return newCachedCloudMetadata(source, func() string { return "example.com" })
}

func TestCloudMetadataSharesOneFetchAcrossConcurrentCallers(t *testing.T) {
	source := &countingCloudMetadata{release: make(chan struct{})}
	cache := newTestCloudMetadata(source)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := cache.Devices(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	// Wait until the first fetch is in flight, then let it finish.
	for source.deviceCalls.Load() == 0 {
		runtime.Gosched()
	}
	close(source.release)
	wg.Wait()

	if _, err := cache.Devices(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := source.deviceCalls.Load(); got != 1 {
		t.Fatalf("device fetches = %d, want 1", got)
	}
}

func TestCloudMetadataCachesErrorsBrieflyButNotCancellation(t *testing.T) {
	source := &countingCloudMetadata{deviceErr: errors.New("rate limited")}
	cache := newTestCloudMetadata(source)
	for range 3 {
		if _, err := cache.Devices(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	}
	if got := source.deviceCalls.Load(); got != 1 {
		t.Fatalf("device fetches after errors = %d, want 1 (errors are cached briefly)", got)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.DevicePostureAttributes(canceled, "n1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, err := cache.DevicePostureAttributes(context.Background(), "n1"); err != nil {
		t.Fatalf("cancellation was cached: %v", err)
	}
	if got := source.postureCalls.Load(); got != 2 {
		t.Fatalf("posture fetches = %d, want 2", got)
	}
}

func TestCloudMetadataInvalidateRefetches(t *testing.T) {
	source := &countingCloudMetadata{}
	cache := newTestCloudMetadata(source)
	_, _ = cache.Devices(context.Background())
	cache.Invalidate()
	_, _ = cache.Devices(context.Background())
	if got := source.deviceCalls.Load(); got != 2 {
		t.Fatalf("device fetches = %d, want 2", got)
	}
}
