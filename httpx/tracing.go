package httpx

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func (s *Server) initTracerProvider() error {
	if !s.cfg.Tracing.Enabled {
		// still must set a handler
		s.handler = s.engine
		return nil
	}

	// 1) Propagator (extract traceparent)
	otel.SetTextMapPropagator(buildPropagator(s.cfg.Tracing))

	// 2) Exporter (example: OTLP/HTTP)
	exp, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(s.cfg.Tracing.OTLPEndpoint), // "http://localhost:4318/v1/traces"
	)
	if err != nil {
		return err
	}

	// 3) Resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(s.cfg.Name),
			semconv.ServiceVersion(s.cfg.Version),
		),
	)
	if err != nil {
		return err
	}

	// 4) TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	// 5) Register stopper
	s.registerStopper(func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	// 6) Wrap Gin engine as final handler
	var h http.Handler = s.engine
	h = otelhttp.NewHandler(
		h,
		"http.server",
		otelhttp.WithServerName(s.cfg.Name),
	)
	s.handler = h

	return nil
}

func buildPropagator(cfg TracingConfig) propagation.TextMapPropagator {
	if len(cfg.Propagators) == 0 {
		if cfg.IncludeBaggage {
			return autoprop.NewTextMapPropagator() // tracecontext+baggage
		}
		p, err := autoprop.TextMapPropagator("tracecontext")
		if err == nil {
			return p
		}
		return propagation.TraceContext{}
	}

	p, err := autoprop.TextMapPropagator(cfg.Propagators...)
	if err != nil {
		p2, err2 := autoprop.TextMapPropagator("tracecontext")
		if err2 == nil {
			return p2
		}
		return propagation.TraceContext{}
	}
	return p
}
