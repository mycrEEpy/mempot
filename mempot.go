package mempot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Cache holds the data you want to cache in memory.
type Cache struct {
	mut  sync.RWMutex
	data map[string]Item

	defaultTTL      time.Duration
	cleanupInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
}

// Item is a unit of data which can be cached and has an expiration as Unix epoch.
type Item struct {
	Data any
	TTL  int64
}

// Expired returns true if the data of the Item has expired.
func (i *Item) Expired() bool {
	return time.Now().Unix() > i.TTL
}

// Option can alter the behavior of a Cache.
type Option func(*Cache)

// New create a new Cache instance.
func New(opts ...Option) *Cache {
	c := &Cache{
		data:            make(map[string]Item),
		defaultTTL:      time.Minute * 15,
		cleanupInterval: time.Minute * 5,
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	for _, opt := range opts {
		opt(c)
	}

	go c.cleanup()

	return c
}

// WithDefaultTTL changes the default time-to-live for an Item in the Cache.
// Default is 15m.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Cache) {
		c.defaultTTL = ttl
	}
}

// WithCleanupInterval changes the default interval at which expired Items are removed from the Cache.
// Default is 5m.
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Cache) {
		c.cleanupInterval = interval
	}
}

// WithContext adds a custom context for the Cache.
// If the context is canceled, the cleanup ticker will stop.
func WithContext(ctx context.Context) Option {
	return func(c *Cache) {
		c.ctx = ctx
	}
}

// Set will add an Item to the Cache with the default time-to-live.
func (c *Cache) Set(key string, value any) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL will add an Item to the Cache with the given time-to-live.
func (c *Cache) SetWithTTL(key string, data any, ttl time.Duration) {
	c.mut.Lock()
	c.data[key] = Item{Data: data, TTL: time.Now().Add(ttl).Unix()}
	c.mut.Unlock()
}

// Get returns an Item and true if the Item was found in the Cache and has not been expired.
// An empty Item and false is returned when the Item was not found or has been expired.
func (c *Cache) Get(key string) (Item, bool) {
	c.mut.RLock()
	item, ok := c.data[key]
	c.mut.RUnlock()

	if item.Expired() {
		return Item{}, false
	}

	return item, ok
}

// QueryFunc is a function to retrieve data which will be put into the Cache.
type QueryFunc func(key string) (any, error)

// Remember tries to get the Item from the Cache, if the Item is not found or expired QueryFunc is called
// to retrieve the data from source and put it into the Cache.
func (c *Cache) Remember(key string, query QueryFunc) (Item, error) {
	return c.RememberWithTTL(key, query, c.defaultTTL)
}

// RememberWithTTL tries to get the Item from the Cache, if the Item is not found or expired QueryFunc is called
// to retrieve the data from source and put it into the Cache with the given time-to-live.
func (c *Cache) RememberWithTTL(key string, query QueryFunc, ttl time.Duration) (Item, error) {
	item, ok := c.Get(key)
	if ok {
		return item, nil
	}

	data, err := query(key)
	if err != nil {
		return Item{}, fmt.Errorf("failed to query data: %w", err)
	}

	c.SetWithTTL(key, data, ttl)

	return Item{Data: data, TTL: time.Now().Add(c.defaultTTL).Unix()}, nil
}

// Delete removes an Item from the Cache.
func (c *Cache) Delete(key string) {
	c.mut.Lock()
	delete(c.data, key)
	c.mut.Unlock()
}

// Reset removes all Items from the Cache.
func (c *Cache) Reset() {
	c.mut.Lock()
	c.data = make(map[string]Item)
	c.mut.Unlock()
}

// Cancel will cancel the default Context of the Cache which stops the cleanup ticker.
// This only has an effect when the Cache has not been created with a custom context.
func (c *Cache) Cancel() {
	c.cancel()
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.cleanupInterval)

	for {
		select {
		case <-c.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			toBeDeleted := make([]string, 0)

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
