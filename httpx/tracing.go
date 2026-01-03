package httpx

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (s *Server) initTracing() error {
	// base handler is Gin engine
	var h http.Handler = s.engine

	// optionally wrap with tracing
	if s.cfg.EnableTracing {
		h = otelhttp.NewHandler(
			h,
			"http.server",
			otelhttp.WithServerName(s.cfg.Name),
		)
	}

	s.handler = h
	return nil
}
