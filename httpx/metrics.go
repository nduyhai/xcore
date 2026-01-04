package httpx

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func (s *Server) initMetrics() error {
	// Metrics endpoint (Prometheus scrape)
	if !s.cfg.EnableMetrics {
		return nil
	}
	// Create Prometheus exporter (OTel Metric Reader)
	exp, err := otelprom.New()
	if err != nil {
		return err
	}

	// Resource (service identity)
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

	// MeterProvider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exp),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	reg := prometheus.NewRegistry()
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	s.engine.GET(s.cfg.MetricsPath, gin.WrapH(h))

	// Register shutdown hook (important!)
	s.registerStopper(func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	return nil
}
