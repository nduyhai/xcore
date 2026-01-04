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
	httpSrv *http.Server

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
			GinMode:           gin.ReleaseMode,
			Profiling: ProfilingConfig{
				Enabled:    false,
				TagByRoute: true,
			},
			Pprof: PprofConfig{
				Enabled: false,
				Prefix:  "/debug/pprof",
			},
			Tracing: TracingConfig{
				Enabled:      false,
				OTLPEndpoint: "http://localhost:4318/v1/traces",
			},
			Metrics: MetricsConfig{
				Enabled:                false,
				Path:                   "/metrics",
				EnableGoCollector:      true,
				EnableProcessCollector: true,
			},
			Obs: ObservabilityConfig{
				Mode:    ObsModeManual,
				Enabled: false,
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
	stopCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	gin.SetMode(s.cfg.GinMode)

	r := gin.New()
	r.Use(gin.Recovery())

	s.engine = r

	// run init pipeline
	for _, init := range s.inits {
		if err := init(s); err != nil {
			_ = s.stopAll(stopCtx) // best-effort cleanup
			return err
		}
	}

	if s.handler == nil {
		s.handler = s.engine
	}

	s.httpSrv = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.handler,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
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
	s.addInit((*Server).initObservability) //Observability
	s.addInit((*Server).initPprof)         // /debug/pprof route or debug server
	s.addInit((*Server).initProfiling)     // pyroscope continuous profiling
	s.addInit((*Server).initRouters)       // routers

}
func (s *Server) registerStopper(f stopFn) {
	s.stoppers = append(s.stoppers, f)
}

func (s *Server) initRouters() error {
	// Register routes from modules
	for _, fn := range s.routeFns {
		fn(s.engine)
	}
	for _, ri := range s.engine.Routes() {
		s.log.Info("route", "method", ri.Method, "path", ri.Path)
	}

	return nil
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
