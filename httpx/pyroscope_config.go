package httpx

type ProfilingConfig struct {
	Enabled       bool
	ServerAddress string            // e.g. http://pyroscope.monitoring:4040
	Tags          map[string]string // env, version, pod, etc
	TagByRoute    bool              // add gin middleware to tag by method+path
	MutexRate     int
	BlockRate     int
}

type ProfilingOption func(*ProfilingConfig)

func WithProfilingTags(tags map[string]string) ProfilingOption {
	return func(c *ProfilingConfig) {
		// copy to avoid external mutation
		if tags == nil {
			return
		}
		c.Tags = make(map[string]string, len(tags))
		for k, v := range tags {
			c.Tags[k] = v
		}
	}
}

func WithProfilingTagByRoute(enabled bool) ProfilingOption {
	return func(c *ProfilingConfig) { c.TagByRoute = enabled }
}

func WithProfilingMutexRate(rate int) ProfilingOption {
	return func(c *ProfilingConfig) { c.MutexRate = rate }
}

func WithProfilingBlockRate(rate int) ProfilingOption {
	return func(c *ProfilingConfig) { c.BlockRate = rate }
}

func NewProfilingConfig(serverAddr string, opts ...ProfilingOption) ProfilingConfig {
	cfg := ProfilingConfig{
		Enabled:       true,
		ServerAddress: serverAddr,
		TagByRoute:    true, // default on
		Tags:          map[string]string{},
	}

	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
