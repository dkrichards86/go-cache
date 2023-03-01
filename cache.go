package cache

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Item is a value in the cache
type Item struct {
	Value      interface{}
	Expiration time.Time
}

// Expired returns true if the item has expired.
func (me *Item) Expired(now time.Time) bool {
	if me.Expiration.IsZero() {
		return false
	}

	return me.Expiration.Before(now)
}

// Cache is an interface for our cache
type Cache interface {
	Get(key string) (*Item, error)
	Set(key string, value *Item)
}

type simpleCache struct {
	store Adapter

	ttl time.Duration

	entries map[string]interface{}
}

func (me *simpleCache) fetchAndStore(key string) (*Item, error) {
	value, err := me.store.Query(key)
	if err != nil {
		return nil, err
	}

	item := value.(*Item)
	me.Set(key, item)
	return item, nil
}

func (me *simpleCache) Get(key string) (*Item, error) {
	value, ok := me.entries[key]
	if !ok {
		return me.fetchAndStore(key)
	}

	item := value.(*Item)
	if item.Expired(time.Now()) {
		return me.fetchAndStore(key)
	}

	return item, nil
}

func (me *simpleCache) Set(key string, item *Item) {
	if me.ttl > 0 {
		item.Expiration = time.Now().Add(me.ttl)
	}

	me.entries[key] = item
}

// NewSimpleCache returns an instance of simpleCache
func NewSimpleCache(store Adapter, ttl time.Duration) *simpleCache {
	entries := make(map[string]interface{})
	return &simpleCache{store: store, entries: entries, ttl: ttl}
}

type concurrentCache struct {
	store Adapter

	ttl time.Duration

	entries *sync.Map
}

func (me *concurrentCache) fetchAndStore(key string) (*Item, error) {
	value, err := me.store.Query(key)
	if err != nil {
		return nil, err
	}

	item := value.(*Item)
	me.Set(key, item)
	return item, nil
}

func (me *concurrentCache) Get(key string) (*Item, error) {
	value, ok := me.entries.Load(key)
	if !ok {
		return me.fetchAndStore(key)
	}

	item := value.(*Item)
	if item.Expired(time.Now()) {
		return me.fetchAndStore(key)
	}

	return item, nil
}

func (me *concurrentCache) Set(key string, item *Item) {
	if me.ttl > 0 {
		item.Expiration = time.Now().Add(me.ttl)
	}

	me.entries.Store(key, item)
}

// NewConcurrentCache returns an instance of concurrentCache
func NewConcurrentCache(store Adapter, ttl time.Duration) *concurrentCache {
	entries := &sync.Map{}
	return &concurrentCache{store: store, entries: entries, ttl: ttl}
}

// lockedCache
type lockedCache struct {
	store Adapter

	ttl time.Duration

	entries sync.Map
	locks   sync.Map
}

func (me *lockedCache) getLock(key string) *sync.Mutex {
	rawLock, _ := me.locks.LoadOrStore(key, &sync.Mutex{})
	return rawLock.(*sync.Mutex)
}

func (me *lockedCache) fetchAndStore(key string) (*Item, error) {
	value, err := me.store.Query(key)
	if err != nil {
		// Check err type here
		return nil, err
	}

	item := value.(*Item)
	me.Set(key, item)
	return item, nil
}

func (me *lockedCache) Get(key string) (*Item, error) {
	lock := me.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	value, ok := me.entries.Load(key)
	if !ok {
		return me.fetchAndStore(key)
	}

	item := value.(*Item)
	if item.Expired(time.Now()) {
		return me.fetchAndStore(key)
	}

	return item, nil
}

func (me *lockedCache) Set(key string, item *Item) {
	if me.ttl > 0 {
		item.Expiration = time.Now().Add(me.ttl)
	}

	me.entries.Store(key, item)
}

// NewLockedCache returns an instance of lockedCache
func NewLockedCache(store Adapter, ttl time.Duration) *lockedCache {
	var entries sync.Map
	var locks sync.Map

	return &lockedCache{store: store, entries: entries, locks: locks, ttl: ttl}
}

// coalescedCache
type coalescedCache struct {
	store Adapter

	ttl time.Duration

	entries sync.Map

	singleflight singleflight.Group
}

func (me *coalescedCache) fetchAndStore(key string) (*Item, error) {
	value, err := me.store.Query(key)
	if err != nil {
		// Check err type here
		return nil, err
	}

	item := value.(*Item)
	me.Set(key, item)
	return item, nil
}

func (me *coalescedCache) Get(key string) (*Item, error) {
	value, err, _ := me.singleflight.Do(key, func() (interface{}, error) {
		item, ok := me.entries.Load(key)
		if !ok {
			return me.fetchAndStore(key)
		}

		if item.(*Item).Expired(time.Now()) {
			return me.fetchAndStore(key)
		}

		return item, nil
	})

	return value.(*Item), err
}

func (me *coalescedCache) Set(key string, item *Item) {
	if me.ttl > 0 {
		item.Expiration = time.Now().Add(me.ttl)
	}

	me.entries.Store(key, item)
}

// NewCoalescedCache returns an instance of coalescedCache
func NewCoalescedCache(store Adapter, ttl time.Duration) *coalescedCache {
	var entries sync.Map
	return &coalescedCache{store: store, entries: entries, ttl: ttl}
}
