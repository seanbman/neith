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

// NewCache creates a typed cache entry for the current client connection.
//
// The connection ID is read from the dispatch details stored in ctx by the
// fcmp middleware. Each connection gets its own cache store, so the same key can
// be reused safely across different browser clients. NewCache returns
// ErrCacheExists if the key already exists for this connection, including when
// the existing cache was created with a different type.
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

// UseCache retrieves a typed cache entry for the current client connection.
//
// The requested type T must match the type used when the cache was created. If
// the key does not exist, UseCache returns ErrCacheNotFound; if the key exists
// with a different type, it returns ErrCacheWrongType.
//
// https://pkg.go.dev/github.com/snburman/fcmp#UseCache
func UseCache[T any](ctx context.Context, key string) (Cache[T], error) {
	dispatch, err := cacheDispatch(ctx)
	if err != nil {
		return Cache[T]{}, err
	}
	return getCache[T](dispatch.ConnID, key)
}

// Set writes a new value into the cache and refreshes its expiry timer.
//
// Passing a positive timeout shorter than config.CacheTimeOut uses that timeout
// for this cache update. Passing 0 keeps the previous timeout when one exists.
// Omitting timeout, passing a negative timeout, or passing a timeout greater
// than config.CacheTimeOut falls back to config.CacheTimeOut. Set also triggers
// OnCacheChange callbacks and starts an expiry watcher for this exact update.
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

// Value returns the latest stored cache value.
//
// If the cache no longer exists or cannot be read as T, Value returns the zero
// value for T. Use UseCache when the caller needs to distinguish those errors.
func (c *Cache[T]) Value() T {
	cache, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil {
		return *new(T)
	}
	return cache.data
}

// Delete removes this cache entry from its connection-local store.
//
// Delete is idempotent from the caller's perspective: deleting a missing cache
// does not return an error, though a missing store may be logged at debug level.
func (c *Cache[T]) Delete() {
	deleteCache(c.storeKey, c.cacheKey)
}

// CreatedAt returns the time recorded when this cache entry was first created.
//
// The timestamp is stored on the cache value itself and is preserved when Set
// updates the cache.
func (c *Cache[T]) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the time recorded when this cache value was last written.
//
// The value changes each time Set successfully persists a new cache value.
func (c *Cache[T]) UpdatedAt() time.Time {
	return c.updatedAt
}

// TimeOut returns the duration after UpdatedAt when this cache entry expires.
//
// Expiry is checked by a background watcher created by Set. The watcher verifies
// the cache has not been updated since it started before deleting anything.
func (c *Cache[T]) TimeOut() time.Duration {
	return c.timeOut
}

// Expiry returns the wall-clock time when the current cache value expires.
//
// Expiry is calculated from UpdatedAt plus TimeOut. It is useful for inspection
// and tests; expiry deletion is handled internally by watchExpiry.
func (c *Cache[T]) Expiry() time.Time {
	return c.updatedAt.Add(c.timeOut)
}

// Record controls whether future cache updates are stored in history.
//
// The flag is persisted into the backing cache entry, so calling Record(true)
// before Set is enough to have later Set calls recorded even though Set reloads
// the current cache state from the store.
func (c *Cache[T]) Record(record bool) {
	c.record = record
	current, err := getCache[T](c.storeKey, c.cacheKey)
	if err != nil {
		return
	}
	current.record = record
	_ = saveCache(c.storeKey, c.cacheKey, current)
}

// History returns the recorded cache values for this cache entry.
//
// The returned map is keyed by the time each value was recorded. The boolean is
// false when no history exists or when a stored history value cannot be asserted
// back to T.
func (c *Cache[T]) History() (map[string]T, bool) {
	return cacheHistory[T](&cacheEvents, c.id())
}

// id returns the stable identifier used by the event registry for this cache.
//
// The backing store is already split by connection ID, but callback and history
// maps are global, so they need a combined connection/key identifier.
func (c *Cache[T]) id() string {
	return cacheID(c.storeKey, c.cacheKey)
}

// watchExpiry deletes a cache entry if this specific update has expired.
//
// Each Set call starts its own watcher. The updatedAt argument lets the watcher
// detect stale goroutines: if another Set call updated the cache after this
// watcher started, the watcher exits without deleting the newer value.
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

// cacheDispatch extracts fcmp dispatch details from a request/event context.
//
// Cache state is scoped to a connection ID, and that connection ID only exists
// inside middleware-populated context. This helper keeps NewCache and UseCache
// consistent about missing-context errors.
func cacheDispatch(ctx context.Context) (dispatchDetails, error) {
	dispatch, ok := dispatchFromContext(ctx)
	if !ok {
		return dispatchDetails{}, ErrCtxMissingDispatch
	}
	return dispatch, nil
}

// resolveCacheTimeout chooses the timeout that should apply to a Set call.
//
// It treats the optional timeout list as "last value wins", preserving a
// previous timeout for explicit 0 values and clamping unsupported values back to
// config.CacheTimeOut.
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

// OnCacheTimeOut registers a callback to run when this cache entry expires.
//
// The callback is keyed to this cache's connection/key pair. It runs after an
// expiry watcher confirms the cache value is still the same update that started
// the watcher.
func OnCacheTimeOut[T any](c Cache[T], f func()) {
	cacheEvents.setTimeout(c.id(), f)
}

// OnCacheChange registers a callback to run after Set writes a cache value.
//
// Updating metadata with helpers like Record uses saveCache and does not trigger
// this callback. Only value-changing Set calls trigger OnCacheChange.
func OnCacheChange[T any](c Cache[T], f func()) {
	cacheEvents.setChange(c.id(), f)
}

// callOnFn dispatches cache lifecycle callbacks for a specific cache value.
//
// On change, it also records the cache value when Record(true) has been enabled.
// Callback execution is delegated to cacheEventRegistry so locks are not held
// while user-provided functions run.
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

// newCacheEventRegistry creates the callback/history registry used by caches.
//
// Tests reset the global registry with this constructor so callbacks and history
// do not leak across cases.
func newCacheEventRegistry() cacheEventRegistry {
	return cacheEventRegistry{
		onchange:  make(map[string]func()),
		ontimeout: make(map[string]func()),
		history:   make(map[string]map[string]any),
	}
}

// Delete removes all callback and history entries for a cache identifier.
//
// This is used to clean up callback state when a cache or connection goes away.
func (r *cacheEventRegistry) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.onchange, id)
	delete(r.ontimeout, id)
	delete(r.history, id)
}

// setChange stores the OnCacheChange callback for a cache identifier.
//
// A later registration replaces the previous callback for that cache.
func (r *cacheEventRegistry) setChange(id string, f func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onchange[id] = f
}

// setTimeout stores the OnCacheTimeOut callback for a cache identifier.
//
// A later registration replaces the previous timeout callback for that cache.
func (r *cacheEventRegistry) setTimeout(id string, f func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ontimeout[id] = f
}

// callChange records optional history and runs the change callback.
//
// The callback is copied while the registry is locked, then invoked after the
// lock is released. That prevents user code from blocking other cache registry
// operations or deadlocking if it touches cache state.
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

// callTimeout runs the timeout callback for a cache identifier.
//
// Like callChange, it copies the callback under lock and invokes it after
// releasing the lock so user code cannot hold the registry mutex.
func (r *cacheEventRegistry) callTimeout(id string) {
	r.mu.Lock()
	fn := r.ontimeout[id]
	r.mu.Unlock()

	if fn != nil {
		fn()
	}
}

// cacheHistory returns a typed copy of the recorded values for a cache.
//
// The registry stores history as any because caches are generic. This function
// performs the type assertion back to T and returns false if any entry has the
// wrong type.
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

// recordHistory stores one value in the history map for a cache identifier.
//
// The caller must already hold r.mu. Keeping this helper lock-free avoids
// nested locking when callChange is already inside the critical section.
func (r *cacheEventRegistry) recordHistory(id string, data any) {
	if _, ok := r.history[id]; !ok {
		r.history[id] = make(map[string]any)
	}
	r.history[id][time.Now().String()] = data
}

var sm = newStoreManager()

type storeManager struct {
	mu     sync.Mutex
	stores map[string]*cacheStore
}

type cacheStore struct {
	mu    sync.Mutex
	cache map[string]any
}

// newStoreManager creates an empty manager for connection-local cache stores.
//
// The global store manager is process-local; each store inside it is keyed by a
// connection ID.
func newStoreManager() storeManager {
	return storeManager{
		stores: make(map[string]*cacheStore),
	}
}

// get returns the cache store associated with a connection ID.
//
// The returned cacheStore has its own lock for entry-level reads and writes.
func (m *storeManager) get(key string) (*cacheStore, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	store, ok := m.stores[key]
	return store, ok
}

// ensure returns an existing cache store or creates one for the connection ID.
//
// NewCache uses this to lazily create per-connection storage before writing the
// first cache entry for that client.
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

// delete removes an entire connection-local cache store.
//
// Connection cleanup calls this after the configured cache timeout when a client
// has not reconnected.
func (m *storeManager) delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stores, key)
}

// createCache ensures the connection store exists and persists a new cache.
//
// It intentionally routes through setCache so normal creation participates in
// the same callback path as other writes.
func createCache[T any](c Cache[T]) error {
	sm.ensure(c.storeKey)
	return setCache(c.storeKey, c.cacheKey, c)
}

// setCache persists a cache value and emits the cache change event.
//
// Use saveCache when internal metadata needs to be saved without firing
// OnCacheChange callbacks.
func setCache[T any](storeKey string, cacheKey string, c Cache[T]) error {
	if err := saveCache(storeKey, cacheKey, c); err != nil {
		return err
	}
	callOnFn(onChange, c)
	return nil
}

// saveCache persists a cache value without emitting callbacks.
//
// This is used for internal metadata updates such as Record(true), where callers
// expect the cache behavior to change but not for OnCacheChange to run.
func saveCache[T any](storeKey string, cacheKey string, c Cache[T]) error {
	store, ok := sm.get(storeKey)
	if !ok {
		return ErrStoreNotFound
	}

	store.set(cacheKey, c)
	return nil
}

// getCache reads and type-checks a cache value from the backing store.
//
// Because the store holds values as any, getCache is the boundary that verifies
// the requested generic type matches the stored cache type.
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

// deleteCache removes one cache entry from a connection-local store.
//
// Missing stores are logged at debug level instead of returned as errors because
// Delete is part of cleanup paths where repeated calls should be harmless.
func deleteCache(storeKey string, cacheKey string) {
	store, ok := sm.get(storeKey)
	if !ok {
		config.Logger.Debug("could not delete cache, no such store", "storeKey", storeKey, "cacheKey", cacheKey)
		return
	}
	store.delete(cacheKey)
}

// get reads one value from a cacheStore under the store lock.
//
// The value remains typed as any until getCache asserts it back to Cache[T].
func (s *cacheStore) get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.cache[key]
	return value, ok
}

// set writes one value into a cacheStore under the store lock.
//
// The caller is responsible for deciding whether callbacks should run.
func (s *cacheStore) set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = value
}

// delete removes one value from a cacheStore under the store lock.
//
// Deleting a missing key is harmless and mirrors map delete semantics.
func (s *cacheStore) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, key)
}

// cacheID joins a connection ID and cache key into a callback/history key.
//
// The separator avoids accidental collisions between pairs such as ("ab", "c")
// and ("a", "bc").
func cacheID(storeKey string, cacheKey string) string {
	return storeKey + ":" + cacheKey
}
