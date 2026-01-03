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
}

type PprofConfig struct {
	Enabled bool
	Prefix  string // default: "/debug/pprof"
}
