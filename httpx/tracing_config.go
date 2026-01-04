package httpx

type PropagationFormat string

const (
	PropW3C    PropagationFormat = "w3c"    // traceparent + tracestate
	PropB3     PropagationFormat = "b3"     // X-B3-* (single or multi)
	PropJaeger PropagationFormat = "jaeger" // uber-trace-id
)

type TracingConfig struct {
	Enabled      bool
	OTLPEndpoint string
	// Optional
	Propagation []PropagationFormat
}

func NewTracingConfig(otlpEndpoint string, opts ...TracingOption) TracingConfig {
	cfg := TracingConfig{
		Enabled:      true,
		OTLPEndpoint: otlpEndpoint,

		// default: modern standard
		Propagation: []PropagationFormat{PropW3C},
	}

	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

type TracingOption func(*TracingConfig)

func WithPropagation(formats ...PropagationFormat) TracingOption {
	return func(c *TracingConfig) {
		// default to W3C if empty
		c.Propagation = append([]PropagationFormat{}, formats...)
	}
}

func WithDiableTracing() Option {
	return func(s *Server) {
		s.cfg.Tracing.Enabled = false
	}
}
