package httpx

import "time"

type Config struct {
	Name string
	Addr string

	// Timeouts
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration

	// Features
	EnableTracing bool
	EnableMetrics bool
	MetricsPath   string // default "/metrics"

	// Gin
	GinMode string // gin.ReleaseMode / gin.DebugMode / gin.TestMode

	// profiling
	Profiling ProfilingConfig
	Pprof     PprofConfig
}

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

type PprofConfig struct {
	Enabled bool
	Prefix  string // default: "/debug/pprof"
}

type PprofOption func(*PprofConfig)

func WithPprofPrefix(prefix string) PprofOption {
	return func(c *PprofConfig) { c.Prefix = prefix }
}

func EnablePprof(opts ...PprofOption) PprofConfig {
	cfg := PprofConfig{
		Enabled: true,
		Prefix:  "/debug/pprof",
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func DisablePprof() PprofConfig { return PprofConfig{Enabled: false} }
