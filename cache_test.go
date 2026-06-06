package neith

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

const testCache = "test_cache"

func init() {
	opts := log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "TESTING neith:",
	}
	logOpts = opts
	config = &Config{
		CacheTimeOut: 5 * time.Minute,
		LogLevel:     Debug,
		Logger:       log.NewWithOptions(os.Stderr, logOpts),
	}
	defaultRuntime.config = config
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func _test_context() context.Context {
	dd := dispatchDetails{
		ClientID:  "test_client_id",
		Conn:      &conn{},
		HandlerID: "test_handler_id",
	}
	return context.WithValue(context.Background(), dispatchKey, dd)
}

func TestNewCache(t *testing.T) {

	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"testNewCache", testNewCacheCreate},
		{"testNewCacheErrExists", testNewCacheExists},
		{"testNewCacheValue", testNewCacheValue},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetTestCacheStore()
			c.fn(t)
		})
	}
}

func testNewCacheCreate(t *testing.T) {
	ctx := _test_context()
	initial := testStruct{"test", 20}
	cache, err := NewCache(ctx, "test", initial)
	if err != nil {
		t.Error(err)
	}
	if cache.Value() != initial {
		t.Errorf("expected %v, got %v", initial, cache.Value())
	}
}

func testNewCacheExists(t *testing.T) {
	ctx := _test_context()
	_, err := NewCache(ctx, t.Name(), testStruct{"test", 20})
	if err != nil {
		t.Error(err)
	}

	_, err = NewCache(ctx, t.Name(), true)
	if !errors.Is(err, ErrCacheExists) {
		t.Errorf("expected %v, got %v", ErrCacheWrongType, err)
	}
}

func testNewCacheValue(t *testing.T) {
	ctx := _test_context()
	initial := testStruct{"test", 20}
	cache, err := NewCache(ctx, "test", initial)
	if err != nil {
		t.Error(err)
	}
	if cache.Value() != initial {
		t.Errorf("expected %v, got %v", initial, cache.Value())
	}
}

func TestUseCache(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"testSetValueUseCache", testUseCacheSetValue},
		{"testUseCacheDelete", testUseCacheDelete},
		{"testUseCacheTimeOut", testUseCacheTimeOut},
		{"testUseCacheOnCacheTimeOut", testUseCacheOnCacheTimeOut},
		{"testUseCacheOnChange", testUseCacheOnChange},
		{"testUseCacheRecordHistory", testUseCacheRecordHistory},
		{"testUseCacheErr", testUseCacheErr},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetTestCacheStore()
			c.fn(t)
		})
	}
}

func testUseCacheSetValue(t *testing.T) {
	_, err := NewCache(_test_context(), t.Name(), true)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}
	if !cache.Value() {
		t.Error("expected false, got true")
	}
}

func testUseCacheTimeOut(t *testing.T) {
	cases := []struct {
		value bool
		exp   time.Duration
	}{
		{true, time.Millisecond * 5},
		{true, time.Millisecond * 6},
		{true, time.Millisecond * 7},
		{true, time.Millisecond * 8},
	}

	_, err := NewCache(_test_context(), t.Name(), false)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}

	for _, c := range cases {
		cache.Set(c.value, c.exp)
		time.Sleep(c.exp * 5)
		if cache.Value() == c.value {
			t.Errorf("expected %v, got %v", !c.value, cache.Value())
		}
	}
}

func testUseCacheDelete(t *testing.T) {
	_, err := NewCache(_test_context(), t.Name(), true)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}
	cache.Set(true)
	if !cache.Value() {
		t.Error("expected true, got false")
	}
	cache.Delete()
	if cache.Value() {
		t.Error("expected false, got true")
	}
}

func testUseCacheOnCacheTimeOut(t *testing.T) {
	_, err := NewCache(_test_context(), t.Name(), true)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}
	timedOut := make(chan struct{})
	OnCacheTimeOut(cache, func() {
		close(timedOut)
	})

	timeOut := time.Millisecond * 5
	cache.Set(false, timeOut)
	select {
	case <-timedOut:
	case <-time.After(timeOut * 5):
		t.Fatal("expected timeout callback to run")
	}
	waitCacheDeleted(t, t.Name())
}

func testUseCacheOnChange(t *testing.T) {
	_, err := NewCache(_test_context(), t.Name(), true)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}
	count := 0
	OnCacheChange(cache, func() {
		count++
	})
	for i := 0; i < 10; i++ {
		err := cache.Set(true)
		if err != nil {
			t.Error(err)
		}
	}
	if count != 10 {
		t.Errorf("expected 10, got %d", count)
	}
}

func testUseCacheRecordHistory(t *testing.T) {
	_, err := NewCache(_test_context(), t.Name(), true)
	if err != nil {
		t.Error(err)
	}
	cache, err := UseCache[bool](_test_context(), t.Name())
	if err != nil {
		t.Error(err)
	}

	cache.Record(true)
	if err := cache.Set(false); err != nil {
		t.Error(err)
	}

	history, ok := cache.History()
	if !ok {
		t.Fatal("expected history to be recorded")
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	for _, value := range history {
		if value {
			t.Errorf("expected recorded value false, got %v", value)
		}
	}
}

func testUseCacheErr(t *testing.T) {
	ctx := _test_context()
	_, err := UseCache[bool](ctx, "test")
	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("expected %v, got %v", ErrCacheNotFound, err)
	}
}

type testStruct struct {
	Name string
	Age  int
}

func resetTestCacheStore() {
	defaultRuntime.stores = newStoreManager()
	defaultRuntime.cacheEvents = newCacheEventRegistry()
}

func waitCacheDeleted(t *testing.T, key string) {
	t.Helper()

	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, err := UseCache[bool](_test_context(), key)
		if errors.Is(err, ErrCacheNotFound) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("cache %q was not deleted", key)
}

func BenchmarkUseCache(b *testing.B) {
	ctx := _test_context()

	cases := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	_, err := NewCache(ctx, testCache, false)
	if err != nil {
		b.Error(err)
		b.Fail()
	}

	for _, c := range cases {
		cache, _ := UseCache[bool](ctx, testCache)
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache.Set(c.value)
				_ = cache.Value()
			}
		})
	}
}
