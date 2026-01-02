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
}
