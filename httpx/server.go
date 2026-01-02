package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	cfg      Config
	log      *slog.Logger
	routeFns []Routes

	engine *gin.Engine
	httpS  *http.Server
}

func New(opts ...Option) *Server {
	// defaults
	s := &Server{
		cfg: Config{
			Name:              "http",
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
		},
		log: slog.Default(),
	}

	for _, o := range opts {
		o(s)
	}

	s.build()
	return s
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) build() {
	gin.SetMode(s.cfg.GinMode)

	r := gin.New()
	r.Use(gin.Recovery())

	// Register routes from modules
	for _, fn := range s.routeFns {
		fn(r)
	}

	// Metrics endpoint (Prometheus scrape)
	if s.cfg.EnableMetrics {
		r.GET(s.cfg.MetricsPath, gin.WrapH(promhttp.Handler()))
	}

	// Handler (optionally traced)
	var handler http.Handler = r
	if s.cfg.EnableTracing {
		handler = otelhttp.NewHandler(r, "http.server")
	}

	s.engine = r
	s.httpS = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}
}
