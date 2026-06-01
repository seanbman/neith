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

func NewCache[T any](ctx context.Context, key string, initial T) (Cache[T], error) {
	dispatch, err := cacheDispatch(ctx)
	if err != nil {
		return Cache[T]{}, err
	}

	if _, err := getCache[T](dispatch.ConnID, key); err == nil || errors.Is(err, ErrCacheWrongType) {
		return Cache[T]{}, ErrCacheExists
	}

	cache := Cache[T]{
		data:      initial,
		storeKey:  dispatch.ConnID,
		cacheKey:  key,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		timeOut:   config.CacheTimeOut,
	}

	if err := createCache(cache); err != nil {
		return Cache[T]{}, err
	}
	return cache, nil
}

// UseCache takes a generic type, context, and a key and returns a Cache of the type.
//
// https://pkg.go.dev/github.com/snburman/fcmp#UseCache
func UseCache[T any](ctx context.Context, key string) (Cache[T], error) {
	dispatch, err := cacheDispatch(ctx)
	if err != nil {
		return Cache[T]{}, err
	}
	return getCache[T](dispatch.ConnID, key)
}

// Set sets the value of the cache with a timeout.
//
// Set timeout to 0 or leave empty for default expiry.
func (c *Cache[T]) Set(data T, timeout ...time.Duration) error {
	current, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil && !errors.Is(err, ErrCacheNotFound) {
		return err
	}

	if err == nil {
		c.createdAt = current.createdAt
		c.record = current.record
	}
	if c.createdAt.IsZero() {
		c.createdAt = time.Now()
	}

	c.data = data
	c.timeOut = resolveCacheTimeout(current, timeout...)
	c.updatedAt = time.Now()

	if err := setCache(c.storeKey, c.cacheKey, *c); err != nil {
		return err
	}

	go c.watchExpiry(c.updatedAt)
	return nil
}

// Value returns the current value of the cache.
func (c *Cache[T]) Value() T {
	cache, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil {
		return *new(T)
	}
	return cache.data
}

// Delete removes the cache from the store.
func (c *Cache[T]) Delete() {
	deleteCache(c.storeKey, c.cacheKey)
}

// CreatedAt returns the time the cache was created.
func (c *Cache[T]) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the time the cache was last updated.
func (c *Cache[T]) UpdatedAt() time.Time {
	return c.updatedAt
}

func (c *Cache[T]) TimeOut() time.Duration {
	return c.timeOut
}

// Expiry returns expiry time of the cache.
func (c *Cache[T]) Expiry() time.Time {
	return c.updatedAt.Add(c.timeOut)
}

// Record controls whether future cache updates are stored in history.
func (c *Cache[T]) Record(record bool) {
	c.record = record
	current, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil {
		return
	}
	current.record = record
	_ = saveCache(c.storeKey, c.cacheKey, current)
}

// History returns the recorded cache values.
func (c *Cache[T]) History() (map[string]T, bool) {
	return cacheHistory[T](&cacheEvents, c.id())
}

func (c *Cache[T]) id() string {
	return cacheID(c.storeKey, c.cacheKey)
}

func (c *Cache[T]) watchExpiry(updatedAt time.Time) {
	if c.timeOut <= 0 {
		return
	}

	time.Sleep(c.timeOut)

	current, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil || !current.updatedAt.Equal(updatedAt) || time.Now().Before(current.Expiry()) {
		return
	}

	callOnFn(onTimeOut, current)
	current.Delete()
}

func cacheDispatch(ctx context.Context) (dispatchDetails, error) {
	dispatch, ok := dispatchFromContext(ctx)
	if !ok {
		return dispatchDetails{}, ErrCtxMissingDispatch
	}
	return dispatch, nil
}

func resolveCacheTimeout[T any](current Cache[T], timeout ...time.Duration) time.Duration {
	if len(timeout) == 0 {
		return config.CacheTimeOut
	}

	requested := timeout[len(timeout)-1]
	if requested == 0 && current.timeOut > 0 {
		return current.timeOut
	}
	if requested > 0 && requested < config.CacheTimeOut {
		return requested
	}
	return config.CacheTimeOut
}

// OnCacheTimeOut sets a function to be called when the cache expires.
func OnCacheTimeOut[T any](c Cache[T], f func()) {
	cacheEvents.setTimeout(c.id(), f)
}

// OnCacheChange sets a function to be called when the cache is updated.
func OnCacheChange[T any](c Cache[T], f func()) {
	cacheEvents.setChange(c.id(), f)
}

func callOnFn[T any](on CacheOnFn, c Cache[T]) {
	switch on {
	case onChange:
		cacheEvents.callChange(c.id(), c.data, c.record)
	case onTimeOut:
		cacheEvents.callTimeout(c.id())
	}
}

var cacheEvents = newCacheEventRegistry()

type cacheEventRegistry struct {
	mu        sync.Mutex
	onchange  map[string]func()
	ontimeout map[string]func()
	history   map[string]map[string]any
}

func newCacheEventRegistry() cacheEventRegistry {
	return cacheEventRegistry{
		onchange:  make(map[string]func()),
		ontimeout: make(map[string]func()),
		history:   make(map[string]map[string]any),
	}
}

func (r *cacheEventRegistry) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.onchange, id)
	delete(r.ontimeout, id)
	delete(r.history, id)
}

func (r *cacheEventRegistry) setChange(id string, f func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onchange[id] = f
}

func (r *cacheEventRegistry) setTimeout(id string, f func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ontimeout[id] = f
}

func (r *cacheEventRegistry) callChange(id string, data any, record bool) {
	r.mu.Lock()
	if record {
		r.recordHistory(id, data)
	}
	fn := r.onchange[id]
	r.mu.Unlock()

	if fn != nil {
		fn()
	}
}

func (r *cacheEventRegistry) callTimeout(id string) {
	r.mu.Lock()
	fn := r.ontimeout[id]
	r.mu.Unlock()

	if fn != nil {
		fn()
	}
}

func cacheHistory[T any](r *cacheEventRegistry, id string) (map[string]T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, ok := r.history[id]
	if !ok {
		return make(map[string]T), false
	}

	history := make(map[string]T, len(records))
	for recordedAt, data := range records {
		value, ok := data.(T)
		if !ok {
			return make(map[string]T), false
		}
		history[recordedAt] = value
	}
	return history, true
}

func (r *cacheEventRegistry) recordHistory(id string, data any) {
	if _, ok := r.history[id]; !ok {
		r.history[id] = make(map[string]any)
	}
	r.history[id][time.Now().String()] = data
}

// NOTE: The following is some rewritten logic from package mnemo and will be extracted.

var sm = newStoreManager()

type storeManager struct {
	mu     sync.Mutex
	stores map[string]*cacheStore
}

type cacheStore struct {
	mu    sync.Mutex
	cache map[string]any
}

func newStoreManager() storeManager {
	return storeManager{
		stores: make(map[string]*cacheStore),
	}
}

func (m *storeManager) get(key string) (*cacheStore, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	store, ok := m.stores[key]
	return store, ok
}

func (m *storeManager) ensure(key string) *cacheStore {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, ok := m.stores[key]
	if ok {
		return store
	}

	store = &cacheStore{
		cache: make(map[string]any),
	}
	m.stores[key] = store
	return store
}

func (m *storeManager) delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stores, key)
}

func createCache[T any](c Cache[T]) error {
	sm.ensure(c.storeKey)
	return setCache(c.storeKey, c.cacheKey, c)
}

func setCache[T any](storeKey string, cacheKey string, c Cache[T]) error {
	if err := saveCache(storeKey, cacheKey, c); err != nil {
		return err
	}
	callOnFn(onChange, c)
	return nil
}

func saveCache[T any](storeKey string, cacheKey string, c Cache[T]) error {
	store, ok := sm.get(storeKey)
	if !ok {
		return ErrStoreNotFound
	}

	store.set(cacheKey, c)
	return nil
}

func getCache[T any](storeKey string, cacheKey string) (Cache[T], error) {
	store, ok := sm.get(storeKey)
	if !ok {
		return Cache[T]{}, ErrCacheNotFound
	}

	value, ok := store.get(cacheKey)
	if !ok {
		return Cache[T]{}, ErrCacheNotFound
	}

	cache, ok := value.(Cache[T])
	if !ok {
		return Cache[T]{}, ErrCacheWrongType
	}
	return cache, nil
}

func deleteCache(storeKey string, cacheKey string) {
	store, ok := sm.get(storeKey)
	if !ok {
		config.Logger.Debug("could not delete cache, no such store", "storeKey", storeKey, "cacheKey", cacheKey)
		return
	}
	store.delete(cacheKey)
}

func (s *cacheStore) get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.cache[key]
	return value, ok
}

func (s *cacheStore) set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = value
}

func (s *cacheStore) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, key)
}

func cacheID(storeKey string, cacheKey string) string {
	return storeKey + ":" + cacheKey
}
