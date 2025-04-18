package mempot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultConfig contains all default values for a Cache.
var DefaultConfig = Config{
	DefaultTTL:      time.Minute * 15,
	CleanupInterval: time.Minute * 5,
}

// Config allows to alter the configuration of a Cache.
type Config struct {
	// DefaultTTL is used by Cache.Set for the Item.TTL.
	// If set to 0, the Item will not expire.
	//
	// Default: 15m
	DefaultTTL time.Duration

	// CleanupInterval is used for the Ticker in the cleanup goroutine.
	// If set to 0, no cleanup goroutine will be created.
	//
	// Default: 5m
	CleanupInterval time.Duration
}

// Cache holds the data you want to cache in memory.
type Cache[K comparable, T any] struct {
	mut  sync.RWMutex
	data map[K]Item[T]

	ctx context.Context
	cfg Config
}

// Item is a unit of typed data which can be cached and has an expiration as Unix epoch.
type Item[T any] struct {
	Data T
	TTL  int64
}

// Expired returns true if the data of the Item has expired.
func (i *Item[T]) Expired() bool {
	if i.TTL == 0 {
		return false
	}

	return time.Now().Unix() > i.TTL
}

func newItem[T any](data T, ttl time.Duration) Item[T] {
	if ttl == 0 {
		return Item[T]{Data: data, TTL: 0}
	}

	return Item[T]{Data: data, TTL: time.Now().Add(ttl).Unix()}
}

// NewCache create a new Cache instance with K as key and T as data.
// If the context is canceled, the Cache will stop the cleanup goroutine.
func NewCache[K comparable, T any](ctx context.Context, cfg Config) *Cache[K, T] {
	c := &Cache[K, T]{
		data: make(map[K]Item[T]),
		ctx:  ctx,
		cfg:  DefaultConfig,
	}

	if cfg.DefaultTTL > 0 {
		c.cfg.DefaultTTL = cfg.DefaultTTL
	}

	if cfg.CleanupInterval > 0 {
		c.cfg.CleanupInterval = cfg.CleanupInterval
	}

	if c.cfg.CleanupInterval > 0 {
		go c.cleanup()
	}

	return c
}

// Set will add an Item to the Cache with the default time-to-live.
func (c *Cache[K, T]) Set(key K, value T) {
	c.SetWithTTL(key, value, c.cfg.DefaultTTL)
}

// SetWithTTL will add an Item to the Cache with the given time-to-live.
func (c *Cache[K, T]) SetWithTTL(key K, data T, ttl time.Duration) {
	c.mut.Lock()
	c.data[key] = newItem(data, ttl)
	c.mut.Unlock()
}

// Get returns an Item and true if the Item was found in the Cache and has not been expired.
// An empty Item and false is returned when the Item was not found or has been expired.
func (c *Cache[K, T]) Get(key K) (Item[T], bool) {
	c.mut.RLock()
	item, ok := c.data[key]
	c.mut.RUnlock()

	if item.Expired() {
		return Item[T]{}, false
	}

	return item, ok
}

// QueryFunc is a function to retrieve data which will be put into the Cache.
type QueryFunc[K comparable, T any] func(key K) (T, error)

// Remember tries to get the Item from the Cache, if the Item is not found or expired QueryFunc is called
// to retrieve the data from source and put it into the Cache.
func (c *Cache[K, T]) Remember(key K, query QueryFunc[K, T]) (Item[T], error) {
	return c.RememberWithTTL(key, query, c.cfg.DefaultTTL)
}

// RememberWithTTL tries to get the Item from the Cache, if the Item is not found or expired QueryFunc is called
// to retrieve the data from source and put it into the Cache with the given time-to-live.
func (c *Cache[K, T]) RememberWithTTL(key K, query QueryFunc[K, T], ttl time.Duration) (Item[T], error) {
	item, ok := c.Get(key)
	if ok {
		return item, nil
	}

	data, err := query(key)
	if err != nil {
		return Item[T]{}, fmt.Errorf("failed to query data: %w", err)
	}

	c.SetWithTTL(key, data, ttl)

	return newItem(data, ttl), nil
}

// Delete removes an Item from the Cache.
func (c *Cache[K, T]) Delete(key K) {
	c.mut.Lock()
	delete(c.data, key)
	c.mut.Unlock()
}

// Reset removes all Items from the Cache.
func (c *Cache[K, T]) Reset() {
	c.mut.Lock()
	c.data = make(map[K]Item[T])
	c.mut.Unlock()
}

func (c *Cache[K, T]) cleanup() {
	ticker := time.NewTicker(c.cfg.CleanupInterval)

	for {
		select {
		case <-c.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			toBeDeleted := make([]K, 0)

			c.mut.RLock()
			for key, item := range c.data {
				if item.Expired() {
					toBeDeleted = append(toBeDeleted, key)
				}
			}
			c.mut.RUnlock()

			c.mut.Lock()
			for _, key := range toBeDeleted {
				delete(c.data, key)
			}
			c.mut.Unlock()
		}
	}
}
