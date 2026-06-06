# Cache

The cache API stores typed, per-client-session state. A cache entry belongs to
the current browser client inside the current Neith app runtime, which means two
visitors can use the same cache key without sharing values. Two routes wrapped
with `neith.MiddleWareFn` also get isolated runtimes and cache stores.

Cache functions must be called with a context created by `neith.MiddleWareFn`.
That context contains the client session ID and runtime used to choose the right
cache store.

## Table Of Contents

- [Basic Flow](#basic-flow)
- [`NewCache[T](ctx, key, initial)`](#newcachetctx-key-initial)
- [`UseCache[T](ctx, key)`](#usecachetctx-key)
- [`Set(value, timeout...)`](#setvalue-timeout)
- [`Value()`](#value)
- [`Delete()`](#delete)
- [`CreatedAt()`](#createdat)
- [`UpdatedAt()`](#updatedat)
- [`TimeOut()`](#timeout)
- [`Expiry()`](#expiry)
- [`Record(true)`](#recordtrue)
- [`History()`](#history)
- [`OnCacheChange(cache, fn)`](#oncachechangecache-fn)
- [`OnCacheTimeOut(cache, fn)`](#oncachetimeoutcache-fn)
- [Internal Flow](#internal-flow)

## Basic Flow

```go
func app(ctx context.Context) neith.FnComponent {
	_, err := neith.NewCache(ctx, "count", 0)
	if err != nil && !errors.Is(err, neith.ErrCacheExists) {
		return neith.FnErr(ctx, err)
	}

	return counter(ctx)
}

func counter(ctx context.Context) neith.FnComponent {
	count, err := neith.UseCache[int](ctx, "count")
	if err != nil {
		return neith.FnErr(ctx, err)
	}

	_ = count.Set(count.Value() + 1)

	return neith.NewFn(ctx, neith.HTML(fmt.Sprintf(
		`<button>Clicked %d times</button>`,
		count.Value(),
	))).WithEvents(func(ctx context.Context) neith.FnComponent {
		return counter(ctx)
	}, neith.EventClick)
}
```

## `NewCache[T](ctx, key, initial)`

Creates a typed cache entry for the current client session.

```go
cache, err := neith.NewCache(ctx, "user", User{Name: "Sean"})
if err != nil {
	return neith.FnErr(ctx, err)
}
```

Notes:

- `T` is inferred from `initial`.
- The key is scoped to the current browser client session and app runtime.
- Returns `ErrCtxMissingDispatch` if `ctx` was not created by Neith middleware.
- Returns `ErrCacheExists` if the key already exists, even if the existing value
  has a different type.

Common pattern:

```go
_, err := neith.NewCache(ctx, "count", 0)
if err != nil && !errors.Is(err, neith.ErrCacheExists) {
	return neith.FnErr(ctx, err)
}
```

## `UseCache[T](ctx, key)`

Retrieves a typed cache entry for the current client session.

```go
count, err := neith.UseCache[int](ctx, "count")
if err != nil {
	return neith.FnErr(ctx, err)
}
```

Notes:

- `T` must match the type originally used by `NewCache`.
- Returns `ErrCacheNotFound` when the key does not exist.
- Returns `ErrCacheWrongType` when the key exists with another type.

## `Set(value, timeout...)`

Updates the cache value and refreshes its expiry watcher.

```go
cache, err := neith.UseCache[int](ctx, "count")
if err != nil {
	return neith.FnErr(ctx, err)
}

if err := cache.Set(cache.Value() + 1); err != nil {
	return neith.FnErr(ctx, err)
}
```

With a custom timeout:

```go
_ = cache.Set("draft saved", 10*time.Second)
```

Timeout behavior:

- No timeout argument uses `config.CacheTimeOut`.
- `0` keeps the previous timeout when one exists.
- Positive values shorter than `config.CacheTimeOut` are used.
- Negative values and values greater than `config.CacheTimeOut` fall back to
  `config.CacheTimeOut`.

Side effects:

- Triggers `OnCacheChange`.
- Starts a background expiry watcher.
- If `Record(true)` was enabled, stores the new value in history.

## `Value()`

Returns the latest stored cache value.

```go
current := cache.Value()
```

Notes:

- If the cache is missing or cannot be read as `T`, it returns the zero value
  for `T`.
- Use `UseCache[T]` when you need an error instead of a zero value.

## `Delete()`

Removes the cache entry from the current client-session store.

```go
cache.Delete()
```

Notes:

- Calling `Value()` after `Delete()` returns the zero value.
- Deleting an already-missing cache is harmless for callers.

## `CreatedAt()`

Returns when the cache entry was first created.

```go
created := cache.CreatedAt()
```

Notes:

- This timestamp is preserved across `Set` calls.
- It is useful for debugging or showing state age.

## `UpdatedAt()`

Returns when the cache entry was last written.

```go
updated := cache.UpdatedAt()
```

Notes:

- This changes after successful `Set` calls.
- Expiry watchers use this timestamp to avoid deleting newer values.

## `TimeOut()`

Returns the timeout duration currently attached to the cache.

```go
ttl := cache.TimeOut()
```

Notes:

- The timeout is measured from `UpdatedAt()`.
- It is set by `Set`, or defaults to `config.CacheTimeOut`.

## `Expiry()`

Returns the time the current value should expire.

```go
expiresAt := cache.Expiry()
```

Equivalent to:

```go
cache.UpdatedAt().Add(cache.TimeOut())
```

## `Record(true)`

Enables history recording for future cache updates.

```go
cache.Record(true)
_ = cache.Set(cache.Value() + 1)
```

Notes:

- Call `Record(true)` before `Set`.
- The setting is persisted into the cache entry.
- `Record(false)` disables future history recording.
- Calling `Record` does not trigger `OnCacheChange`.

## `History()`

Returns recorded cache values keyed by update time.

```go
history, ok := cache.History()
if ok {
	for recordedAt, value := range history {
		fmt.Println(recordedAt, value)
	}
}
```

Notes:

- The map key is a timestamp string.
- `ok` is false when no history exists.
- `ok` is also false if a stored value cannot be asserted back to `T`.

## `OnCacheChange(cache, fn)`

Registers a callback that runs after `Set` writes a value.

```go
neith.OnCacheChange(cache, func() {
	fmt.Println("cache changed")
})
```

Notes:

- The callback is tied to the cache's client session, runtime, and key.
- A later registration replaces the previous callback.
- Metadata updates like `Record(true)` do not trigger this callback.

## `OnCacheTimeOut(cache, fn)`

Registers a callback that runs when the cache expires.

```go
neith.OnCacheTimeOut(cache, func() {
	fmt.Println("cache expired")
})
```

Notes:

- Expiry is checked by a watcher started by `Set`.
- If a newer `Set` happens before the watcher wakes up, the stale watcher exits
  and does not delete the newer value.

## Internal Flow

```text
NewCache / UseCache
        |
        v
context dispatch details -> runtime + client session ID -> cache store

Set(value)
        |
        v
save cache value
        |
        v
callOnFn(onChange)
        |
        +--> record history when Record(true)
        |
        +--> run OnCacheChange callback
        |
        v
start expiry watcher
        |
        v
run OnCacheTimeOut and delete cache when expired
```
