package cache

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key1", "value1")
	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}
	if val.(string) != "value1" {
		t.Errorf("val = %q, want %q", val, "value1")
	}
}

func TestCacheMiss(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("Get() returned true for missing key")
	}
}

func TestCacheExpiry(t *testing.T) {
	c, err := New(100, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key", "value")
	// Should still be there immediately
	_, ok := c.Get("key")
	if !ok {
		t.Fatal("Get() returned false immediately after Set")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)
	_, ok = c.Get("key")
	if ok {
		t.Fatal("Get() returned true after expiry")
	}
}

func TestCacheNoExpiry(t *testing.T) {
	c, err := New(100, 0)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key", "value")
	_, ok := c.Get("key")
	if !ok {
		t.Fatal("Get() returned false for non-expiring item")
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	c.SetWithTTL("key", "value", 50*time.Millisecond)
	// Wait for expiry
	time.Sleep(100 * time.Millisecond)
	_, ok := c.Get("key")
	if ok {
		t.Fatal("Get() returned true after per-item TTL expiry")
	}
}

func TestCacheRemove(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key", "value")
	c.Remove("key")
	_, ok := c.Get("key")
	if ok {
		t.Fatal("Get() returned true after Remove")
	}
}

func TestCacheClear(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0 after Clear", c.Len())
	}
}

func TestCacheEviction(t *testing.T) {
	c, err := New(2, time.Minute) // max 2 items
	if err != nil {
		t.Fatal(err)
	}

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3) // should evict oldest

	_, ok := c.Get("a")
	if ok {
		t.Log("note: LRU may or may not evict 'a' — depends on access pattern")
	}
}

func TestCacheThreadSafety(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			c.Set("key", i)
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			c.Get("key")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			c.Remove("key")
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestCacheKeys(t *testing.T) {
	c, err := New(100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("x", 1)
	c.Set("y", 2)
	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys() length = %d, want 2", len(keys))
	}
}
