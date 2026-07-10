package dhcache

import "testing"

func TestCacheInstancesAreIndependentAndShutdownIsIdempotent(t *testing.T) {
	first := NewCache()
	second := NewCache()

	if err := first.Set("key", "first"); err != nil {
		t.Fatalf("set first cache: %v", err)
	}
	if _, ok := second.Get("key"); ok {
		t.Fatal("independent cache instance unexpectedly shared data")
	}

	first.Shutdown()
	first.Shutdown()
	second.Shutdown()
}
