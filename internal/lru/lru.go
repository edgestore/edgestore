package lru

import (
	"container/list"
	"sync"
)

// cache implements an LRU cache.
type Cache struct {
	mu       sync.Mutex
	cache    map[string]*list.Element
	priority *list.List
	maxSize  int
}

type kv struct {
	key   string
	value interface{}
}

func New(size int) *Cache {
	return &Cache{
		maxSize:  size,
		priority: list.New(),
		cache:    make(map[string]*list.Element),
	}
}

func (lru *Cache) Put(key string, value interface{}) {
	if _, ok := lru.Get(key); ok {
		return
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()
	if len(lru.cache) == lru.maxSize {
		last := lru.priority.Remove(lru.priority.Back())
		delete(lru.cache, last.(kv).key)
	}
	lru.priority.PushFront(kv{key: key, value: value})
	lru.cache[key] = lru.priority.Front()
}

func (lru *Cache) Del(key string) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	e := lru.cache[key]
	if e == nil {
		return
	}
	delete(lru.cache, key)
	lru.priority.Remove(e)
}

func (lru *Cache) Get(key string) (interface{}, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if element, ok := lru.cache[key]; ok {
		lru.priority.MoveToFront(element)
		return element.Value.(kv).value, true
	}
	return nil, false
}
