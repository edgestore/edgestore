package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
)

const (
	// Default expiration is set to never expire.
	DefaultExpiration = time.Duration(0)

	// DefaultInMemoryCleanup expiration is set to cleanup every 10 minutes.
	DefaultInMemoryCleanup = 10 * time.Minute
)

type Service interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, expires time.Duration) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Run() error
	Shutdown() error
}

type item struct {
	Value      interface{}
	Expiration *time.Time
}

// Returns true if the item has expired.
func (i *item) Expired() bool {
	if i.Expiration != nil {
		return i.Expiration.Before(time.Now())
	}

	return false
}

// InMemory cache is based on https://github.com/gin-contrib/cache
type InMemory struct {
	mux   *sync.Mutex
	items map[string]*item

	cleanupInterval time.Duration
	stop            chan bool
}

func NewInMemory(defaultExpiration, cleanupInterval time.Duration) *InMemory {
	cache := &InMemory{
		mux:             &sync.Mutex{},
		items:           make(map[string]*item),
		cleanupInterval: cleanupInterval,
		stop:            make(chan bool),
	}
	return cache
}

func (c *InMemory) Get(ctx context.Context, key string, value interface{}) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	v, exists := c.items[key]
	if !exists {
		return errors.E(errors.NotFound)
	}

	if v.Expired() {
		delete(c.items, key)
		return errors.E(errors.NotFound)
	}

	if reflect.TypeOf(v.Value) != reflect.TypeOf(value) {
		return errors.E(errors.Internal, fmt.Sprintf("invalid cast type: got %T, want %T", v.Value, value))
	}

	value = v.Value

	return nil
}

func (c *InMemory) Set(ctx context.Context, key string, value interface{}, expiresIn time.Duration) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	expiration := time.Now().Add(expiresIn)
	c.items[key] = &item{
		Value:      value,
		Expiration: &expiration,
	}

	return nil
}

func (c *InMemory) Delete(ctx context.Context, key string) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.items, key)

	return nil
}

func (c *InMemory) DeleteExpired() {
	c.mux.Lock()
	defer c.mux.Unlock()
	for k, v := range c.items {
		if v.Expired() {
			delete(c.items, k)
		}
	}

}

func (c *InMemory) Flush(ctx context.Context) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.items = make(map[string]*item)
	return nil
}

func (c *InMemory) Run() error {
	runJanitor := func(interval time.Duration, stop chan bool) {
		tick := time.Tick(interval)

		for {
			select {
			case <-tick:
				c.DeleteExpired()
			case <-stop:
				return
			}
		}
	}

	if c.cleanupInterval > 0 {
		go runJanitor(c.cleanupInterval, c.stop)
	}

	return nil
}

func (c *InMemory) Shutdown() error {
	if c.cleanupInterval > 0 {
		c.stop <- true
	}

	return nil
}
