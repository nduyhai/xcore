package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg      Config
	log      *slog.Logger
	routeFns []Routes

	engine  *gin.Engine
	handler http.Handler
	httpS   *http.Server

	inits    []initFn
	stoppers []stopFn
}

type initFn func(*Server) error
type stopFn func(ctx context.Context) error

func New(opts ...Option) (*Server, error) {
	// defaults
	s := &Server{
		cfg: Config{
			Name:              "http",
			Version:           "1.0.0",
			Addr:              ":8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   15 * time.Second,
			EnableTracing:     false,
			EnableMetrics:     false,
			MetricsPath:       "/metrics",
			GinMode:           gin.ReleaseMode,
			Profiling: ProfilingConfig{
				Enabled:    false,
				TagByRoute: true,
			},
			Pprof: PprofConfig{
				Enabled: false,
				Prefix:  "/debug/pprof",
			},
		},
		log: slog.Default(),
	}

	for _, o := range opts {
		o(s)
	}

	// register module init functions (central place)
	s.registerInits()

	return s, s.build()
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) build() error {
	gin.SetMode(s.cfg.GinMode)

	r := gin.New()
	r.Use(gin.Recovery())

	// Register routes from modules
	for _, fn := range s.routeFns {
		fn(r)
	}

	s.engine = r

	stopCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	// run init pipeline
	for _, init := range s.inits {
		if err := init(s); err != nil {
			_ = s.stopAll(stopCtx) // best-effort cleanup
			return err
		}
	}

	s.httpS = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.handler,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}

	for _, ri := range s.engine.Routes() {
		s.log.Info("route", "method", ri.Method, "path", ri.Path)
	}
	return nil
}

func (s *Server) addInit(f initFn) {
	s.inits = append(s.inits, f)
}

func (s *Server) addStopper(f stopFn) {
	s.stoppers = append(s.stoppers, f)
}

func (s *Server) registerInits() {
	// order matters
	s.addInit((*Server).initTracing)   //Tracing
	s.addInit((*Server).initMetrics)   // Metrics
	s.addInit((*Server).initPprof)     // /debug/pprof route or debug server
	s.addInit((*Server).initProfiling) // pyroscope continuous profiling

}
func (s *Server) registerStopper(f stopFn) {
	s.stoppers = append(s.stoppers, f)
}

func (s *Server) stopAll(ctx context.Context) error {
	var errs []error

	for i := len(s.stoppers) - 1; i >= 0; i-- {
		stop := s.stoppers[i]
		if stop == nil {
			continue
		}
		if err := stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
