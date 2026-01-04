package httpx

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	jaegerprop "go.opentelemetry.io/contrib/propagators/jaeger"
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
		otlptracehttp.WithEndpoint(s.cfg.Tracing.OTLPEndpoint), // "localhost:4318"
		otlptracehttp.WithInsecure(),
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
	var props []propagation.TextMapPropagator

	// always safe to include baggage
	props = append(props, propagation.Baggage{})

	// default behavior if not set
	formats := cfg.Propagation
	if len(formats) == 0 {
		formats = []PropagationFormat{PropW3C}
	}

	for _, f := range formats {
		switch f {
		case PropW3C:
			props = append(props, propagation.TraceContext{})
		case PropB3:
			// B3 supports both single and multi header via New()
			props = append(props, b3prop.New())
		case PropJaeger:
			props = append(props, jaegerprop.Jaeger{})
		}
	}

	return propagation.NewCompositeTextMapPropagator(props...)
}
