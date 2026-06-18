package tailorcore

import "testing"

func TestPolicyCacheWarmingTracksOverlappingWarmups(t *testing.T) {
	service := &Service{}
	if service.PolicyCacheWarming() {
		t.Fatal("new service unexpectedly reports cache warmup")
	}

	service.warmups.Add(2)
	if !service.PolicyCacheWarming() {
		t.Fatal("active warmups were not reported")
	}
	service.warmups.Add(-1)
	if !service.PolicyCacheWarming() {
		t.Fatal("warmup state cleared while another warmup remained")
	}
	service.warmups.Add(-1)
	if service.PolicyCacheWarming() {
		t.Fatal("completed warmups remain active")
	}
}
