package httpx

type MetricsConfig struct {
	Enabled bool
	Path    string // default "/metrics"

	// optional knobs (future-proof)
	EnableGoCollector      bool
	EnableProcessCollector bool
}

func NewMetricConfig(opts ...MetricsOption) MetricsConfig {
	cfg := MetricsConfig{
		Enabled:                true,
		Path:                   "/metrics",
		EnableGoCollector:      true,
		EnableProcessCollector: true,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func DisableMetrics() MetricsConfig {
	return MetricsConfig{Enabled: false}
}

type MetricsOption func(*MetricsConfig)

func WithMetricsPath(path string) MetricsOption {
	return func(c *MetricsConfig) { c.Path = path }
}

func WithGoCollector(enabled bool) MetricsOption {
	return func(c *MetricsConfig) { c.EnableGoCollector = enabled }
}

func WithProcessCollector(enabled bool) MetricsOption {
	return func(c *MetricsConfig) { c.EnableProcessCollector = enabled }
}
