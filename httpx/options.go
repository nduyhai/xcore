package httpx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type Option func(*Server)

func WithConfig(cfg Config) Option {
	return func(s *Server) { s.cfg = mergeConfig(s.cfg, cfg) }
}

func WithName(name string) Option {
	return func(s *Server) { s.cfg.Name = name }
}

func WithAddr(addr string) Option {
	return func(s *Server) { s.cfg.Addr = addr }
}

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		if l != nil {
			s.log = l
		}
	}
}

func WithGinMode(mode string) Option {
	return func(s *Server) { s.cfg.GinMode = mode }
}

func WithGinRelease() Option { return WithGinMode(gin.ReleaseMode) }

func WithTimeouts(readHeader, read, write, idle, shutdown time.Duration) Option {
	return func(s *Server) {
		if readHeader > 0 {
			s.cfg.ReadHeaderTimeout = readHeader
		}
		if read > 0 {
			s.cfg.ReadTimeout = read
		}
		if write > 0 {
			s.cfg.WriteTimeout = write
		}
		if idle > 0 {
			s.cfg.IdleTimeout = idle
		}
		if shutdown > 0 {
			s.cfg.ShutdownTimeout = shutdown
		}
	}
}

func WithMetrics(path string) Option {
	return func(s *Server) {
		s.cfg.EnableMetrics = true
		if path != "" {
			s.cfg.MetricsPath = path
		}
	}
}

func WithTracing(enable bool) Option {
	return func(s *Server) { s.cfg.EnableTracing = enable }
}

func WithRoutes(fn Routes) Option {
	return func(s *Server) {
		if fn != nil {
			s.routeFns = append(s.routeFns, fn)
		}
	}
}

func mergeConfig(base, in Config) Config {
	// “in” overrides only non-zero / non-empty fields
	if in.Name != "" {
		base.Name = in.Name
	}
	if in.Addr != "" {
		base.Addr = in.Addr
	}
	if in.ReadHeaderTimeout > 0 {
		base.ReadHeaderTimeout = in.ReadHeaderTimeout
	}
	if in.ReadTimeout > 0 {
		base.ReadTimeout = in.ReadTimeout
	}
	if in.WriteTimeout > 0 {
		base.WriteTimeout = in.WriteTimeout
	}
	if in.IdleTimeout > 0 {
		base.IdleTimeout = in.IdleTimeout
	}
	if in.ShutdownTimeout > 0 {
		base.ShutdownTimeout = in.ShutdownTimeout
	}
	if in.MetricsPath != "" {
		base.MetricsPath = in.MetricsPath
	}
	if in.GinMode != "" {
		base.GinMode = in.GinMode
	}
	// booleans: if caller wants explicit control, they should use WithTracing/WithMetrics
	return base
}
