package neith

type runtime struct {
	handlers       handlerPool
	sessions       clientSessionRegistry
	eventListeners eventListeners
	stores         storeManager
	cacheEvents    cacheEventRegistry
	config         *Config
}

var defaultRuntime *runtime

func newRuntime(config *Config) *runtime {
	if config == nil {
		config = defaultConfig()
	}
	return &runtime{
		handlers:       newHandlerPool(),
		sessions:       newClientSessionRegistry(),
		eventListeners: newEventListeners(),
		stores:         newStoreManager(),
		cacheEvents:    newCacheEventRegistry(),
		config:         config,
	}
}

func (r *runtime) Config() *Config {
	if r == nil || r.config == nil {
		return config
	}
	return r.config
}

func runtimeFromDispatch(details dispatchDetails) *runtime {
	if details.Runtime != nil {
		return details.Runtime
	}
	return defaultRuntime
}
