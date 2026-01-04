package httpx

type TracingConfig struct {
	Enabled      bool
	OTLPEndpoint string
	// Optional
	Propagators []string
}

func NewTracingConfig(otlpEndpoint string, opts ...TracingOption) TracingConfig {
	cfg := TracingConfig{
		Enabled:      true,
		OTLPEndpoint: otlpEndpoint,

		// default: modern standard
		Propagators: []string{"tracecontext"},
	}

	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

type TracingOption func(*TracingConfig)

func WithPropagation(formats ...string) TracingOption {
	return func(c *TracingConfig) {
		// default to W3C if empty
		c.Propagators = append([]string{}, formats...)
	}
}

func WithDiableTracing() Option {
	return func(s *Server) {
		s.cfg.Tracing.Enabled = false
	}
}
