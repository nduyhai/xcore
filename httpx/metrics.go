package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) initMetrics() error {
	// Metrics endpoint (Prometheus scrape)
	if s.cfg.EnableMetrics {
		s.engine.GET(s.cfg.MetricsPath, gin.WrapH(promhttp.Handler()))
	}
	return nil
}
