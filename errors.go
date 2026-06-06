package neith

type DispatchError string

func (e DispatchError) Error() string {
	return string(e)
}

const (
	ErrCtxMissingDispatch DispatchError = "context missing dispatch"
	ErrNoClientConnection DispatchError = "no connection to client"
	ErrConnectionNotFound DispatchError = "connection not found"
	ErrConnectionFailed   DispatchError = "connection failed"
	ErrCtxMissingEvent    DispatchError = "context missing event"
)

type CacheError string

func (e CacheError) Error() string {
	return string(e)
}

const (
	ErrCacheNotFound  CacheError = "cache not found"
	ErrStoreNotFound  CacheError = "cache store not found; create cache first"
	ErrCacheWrongType CacheError = "cache wrong type"
	ErrCacheExists    CacheError = "cache already exists; delete existing cache first"
)
