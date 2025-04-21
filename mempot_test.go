package mempot

import (
	"context"
	"errors"
	"testing"
	"time"
)

const key = "foo"
const data = "bar"

func setupCache(ttlSec, intervalSec int) (*Cache[string, string], context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	cache := NewCache[string, string](ctx, Config{
		DefaultTTL:      time.Second * time.Duration(ttlSec),
		CleanupInterval: time.Second * time.Duration(intervalSec),
	})

	return cache, cancel
}

func TestCacheSetGet(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

	cache.Set(key, data)

	item, ok := cache.Get(key)
	if !ok {
		t.Error("item not found")
	}

	if item.Data != data {
		t.Errorf("got %s, want %s", item.Data, data)
	}
}

func TestCacheExpire(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

	cache.SetWithTTL(key, data, time.Millisecond*50)

	time.Sleep(time.Millisecond * 100)

	_, ok := cache.Get(key)
	if ok {
		t.Error("item should have expired")
	}

	// wait for cleanup
	time.Sleep(time.Millisecond * 1100)

	_, ok = cache.Get(key)
	if ok {
		t.Error("item still exists after cleanup")
	}
}

func TestCacheDelete(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

	cache.Set(key, data)

	cache.Delete(key)

	_, ok := cache.Get(key)
	if ok {
		t.Error("item still exists after delete")
	}
}

func TestCacheReset(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

	cache.Set(key, data)
	cache.Reset()

	_, ok := cache.Get(key)
	if ok {
		t.Error("item still exists after reset")
	}
}

func TestCacheRemember(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

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
		t.Errorf("failed to remember item first time: %s", err)
	}

	_, err = cache.Remember(key, func(key string) (string, error) {
		return data, nil
	})
	if err != nil {
		t.Errorf("failed to remember item second time: %s", err)
	}
}

func TestCacheNonExpiring(t *testing.T) {
	cache, cancel := setupCache(1, 1)
	defer cancel()

	cache.SetWithTTL(key, data, 0)

	time.Sleep(time.Millisecond * 1100)

	_, ok := cache.Get(key)
	if !ok {
		t.Error("non expiring item not found")
	}
}
