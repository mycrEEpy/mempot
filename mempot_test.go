package mempot

import (
	"context"
	"errors"
	"testing"
	"time"
)

const key = "foo"
const data = "bar"

func TestCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cache := NewCache[string, string](ctx, Config{
		DefaultTTL:      30 * time.Second,
		CleanupInterval: 4 * time.Second,
	})

	cache.SetWithTTL(key, "bar", 2*time.Second)

	item, ok := cache.Get(key)
	if !ok {
		t.Error("item not found")
	}

	if item.Data != data {
		t.Errorf("got %s, want %s", item.Data, data)
	}

	// wait for cleanup
	time.Sleep(6 * time.Second)

	_, ok = cache.Get(key)
	if ok {
		t.Error("item still exists after cleanup")
	}

	cache.Set(key, data)
	cache.Delete(key)

	_, ok = cache.Get(key)
	if ok {
		t.Error("item still exists after delete")
	}

	cache.Set(key, data)
	cache.Reset()

	_, ok = cache.Get(key)
	if ok {
		t.Error("item still exists after delete all")
	}

	_, err := cache.Remember(key, func(key string) (string, error) {
		return "", errors.New("data not available")
	})
	if err == nil {
		t.Error("QueryFunc failed but Remember did not return an error")
	}

	_, err = cache.Remember(key, func(key string) (string, error) {
		return data, nil
	})
	if err != nil {
		t.Errorf("failed to remember item: %s", err)
	}

	cancel()

	time.Sleep(1 * time.Second)
}
