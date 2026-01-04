package httpx

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
