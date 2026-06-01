package fcmp

import (
	"context"
	"errors"
	"sync"
	"time"
)

type CacheOnFn string

const (
	onChange  CacheOnFn = "onchange"
	onTimeOut CacheOnFn = "ontimeout"
)

type Cache[T any] struct {
	data      T
	storeKey  string
	cacheKey  string
	createdAt time.Time
	updatedAt time.Time
	timeOut   time.Duration
	record    bool
}

// Set sets the value of the cache with a timeout
//
// Set timeout to 0 or leave empty for default expiry.
func (c *Cache[T]) Set(data T, timeout ...time.Duration) error {
	c.data = data
	cache, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil && !errors.Is(err, ErrCacheNotFound) {
		return err
	}
	if err == nil {
		c.createdAt = cache.createdAt
		c.record = cache.record
	}

	for _, t := range timeout {
		switch t {
		case 0:
			if errors.Is(err, ErrCacheNotFound) {
				c.timeOut = config.CacheTimeOut
			} else {
				c.timeOut = cache.timeOut
			}
		default:
			if t > 0 && t < config.CacheTimeOut {
				c.timeOut = t
			} else {
				c.timeOut = config.CacheTimeOut
			}
		}
	}
	if len(timeout) == 0 {
		c.timeOut = config.CacheTimeOut
	}

	c.updatedAt = time.Now()

	go c.watchExpiry()
	err = setCache(c.storeKey, c.cacheKey, *c)
	return err
}

// Value returns the current value of the cache
func (c *Cache[T]) Value() T {
	cache, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil {
		return *new(T)
	}
	return cache.data
}

// Delete removes the cache from the store
func (c *Cache[T]) Delete() {
	deleteCache(c.storeKey, c.cacheKey)
}

// CreatedAt returns the time the cache was created
func (c *Cache[T]) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the time the cache was last updated
func (c *Cache[T]) UpdatedAt() time.Time {
	return c.updatedAt
}

func (c *Cache[T]) TimeOut() time.Duration {
	return c.timeOut
}

// Expiry returns expiry time of the cache
func (c *Cache[T]) Expiry() time.Time {
	return c.updatedAt.Add(c.timeOut)
}

// History returns the history of the cache
func (c *Cache[T]) Record(r bool) {
	c.record = r
}

// GetHistory returns the history of the cache
func (c *Cache[T]) History() (map[string]T, bool) {
	c.record = false
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	h, ok := onfns.history[c.storeKey+c.cacheKey]
	if !ok {
		return make(map[string]T), false
	}
	history := make(map[string]T)
	for k, v := range h {
		t, ok := v.(T)
		if !ok {
			return make(map[string]T), false
		}
		history[k] = t
	}
	return history, true
}

// watchExpiry watches the cache for expiry and, when it expires,
// calls the onTimeOut function and deletes the cache.
func (c *Cache[T]) watchExpiry() {
	if c.timeOut <= 0 {
		return
	}
	for {
		time.Sleep(c.timeOut)
		if time.Now().After(c.Expiry()) {
			callOnFn(onTimeOut, *c)
			c.Delete()
			break
		}
	}
}

func NewCache[T any](ctx context.Context, key string, initial T) (c Cache[T], err error) {
	empty := Cache[T]{}
	dispatch, ok := dispatchFromContext(ctx)
	if !ok {
		return empty, ErrCtxMissingDispatch
	}
	// Check if the cache already exists
	_, err = getCache[T](dispatch.ConnID, key)
	if err == nil {
		return empty, ErrCacheExists
	}
	if errors.Is(err, ErrCacheWrongType) {
		return empty, ErrCacheExists
	}

	// Create a new cache
	cache, ok := newCache(dispatch.ConnID, key, initial)
	if !ok {
		return empty, ErrStoreNotFound
	}
	// Set the initial value of the cache
	cache.data = initial
	err = setCache(dispatch.ConnID, key, cache)
	if err != nil {
		return empty, err
	}
	return cache, nil
}

// UseCache takes a generic type, context, and a key and returns a Cache of the type
//
// https://pkg.go.dev/github.com/snburman/fcmp#UseCache
func UseCache[T any](ctx context.Context, key string) (c Cache[T], err error) {
	empty := Cache[T]{}
	dispatch, ok := dispatchFromContext(ctx)
	if !ok {
		return empty, ErrCtxMissingDispatch
	}
	cache, err := getCache[T](dispatch.ConnID, key)
	if err != nil {
		return empty, err
	}
	return cache, nil
}

// User set callback functions for cache events

type _onfns struct {
	mu        sync.Mutex
	onchange  map[string]any
	ontimeout map[string]any
	history   map[string]map[string]any
}

var onfns = _onfns{
	onchange:  make(map[string]any),
	ontimeout: make(map[string]any),
	history:   make(map[string]map[string]any),
}

func (o *_onfns) Delete(id string) {
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	delete(onfns.onchange, id)
	delete(onfns.ontimeout, id)
}

func (o *_onfns) AddHistory(id string, data any) {
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	if _, ok := onfns.history[id]; !ok {
		onfns.history[id] = make(map[string]any)
	}
	onfns.history[id][time.Now().String()] = data
}

// OnCacheTimeOut sets a function to be called when the cache expires
func OnCacheTimeOut[T any](c Cache[T], f func()) {
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	onfns.ontimeout[c.storeKey+c.cacheKey] = f
}

// OnChange sets a function to be called when the cache is updated
func OnCacheChange[T any](c Cache[T], f func()) {
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	onfns.onchange[c.storeKey+c.cacheKey] = f
}

func callOnFn[T any](on CacheOnFn, c Cache[T]) {
	onfns.mu.Lock()
	defer onfns.mu.Unlock()
	switch on {
	case onChange:
		if c.record {
			id := c.storeKey + c.cacheKey
			if _, ok := onfns.history[id]; !ok {
				onfns.history[id] = make(map[string]any)
			}
			onfns.history[id][time.Now().String()] = c
		}
		if f, ok := onfns.onchange[c.storeKey+c.cacheKey]; ok {
			fn, ok := f.(func())
			if !ok {
				return
			}
			fn()
		}
	case onTimeOut:
		if f, ok := onfns.ontimeout[c.storeKey+c.cacheKey]; ok {
			fn, ok := f.(func())
			if !ok {
				return
			}
			fn()
		}
	}
}

// NOTE: The following is some rewritten logic from package mnemo and will be extracted.

var sm = storeManager{
	stores: make(map[interface{}]*store),
}

type storeManager struct {
	mu     sync.Mutex
	stores map[interface{}]*store
}

type store struct {
	mu    sync.Mutex
	cache map[any]any
}

func (sm *storeManager) get(key interface{}) (*store, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.stores[key]
	if !ok {
		return nil, false
	}
	return s, true
}

func (sm *storeManager) set(key interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stores[key] = &store{
		cache: make(map[any]any),
	}
}

func (sm *storeManager) delete(key interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.stores, key)
}

func setCache[T any](storeKey string, cacheKey string, c Cache[T]) error {
	s, ok := sm.get(storeKey)
	if !ok {
		return ErrStoreNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[cacheKey] = c
	callOnFn(onChange, c)
	return nil
}

func newCache[T any](storeKey string, cacheKey string, cache T) (Cache[T], bool) {
	s, ok := sm.get(storeKey)
	if !ok {
		sm.set(storeKey)
		s, ok = sm.get(storeKey)
		if !ok {
			config.Logger.Debug("failed to create new cache store", "storeKey", storeKey, "cacheKey", cacheKey)
			return Cache[T]{}, false
		}
		s.cache = make(map[any]any)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &Cache[any]{
		storeKey:  storeKey,
		cacheKey:  cacheKey,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		data:      cache,
	}
	s.cache[cacheKey] = c

	copy := Cache[T]{
		storeKey:  storeKey,
		cacheKey:  cacheKey,
		createdAt: c.createdAt,
		updatedAt: c.updatedAt,
		data:      cache,
	}
	return copy, true
}

func getCache[T any](storeKey string, cacheKey string) (Cache[T], error) {
	cache := Cache[T]{}
	s, ok := sm.get(storeKey)
	if !ok {
		return cache, ErrCacheNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cache[cacheKey]
	if !ok {
		return cache, ErrCacheNotFound
	}

	d, ok := c.(Cache[T])
	if !ok {
		return cache, ErrCacheWrongType
	}

	return d, nil
}

func deleteCache(storeKey string, cacheKey string) {
	s, ok := sm.get(storeKey)
	if !ok {
		config.Logger.Debug("could not delete cache, no such store", "storeKey", storeKey, "cacheKey", cacheKey)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, cacheKey)
}
