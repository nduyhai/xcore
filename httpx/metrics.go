package httpx

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func (s *Server) initMetrics() error {
	// Metrics endpoint (Prometheus scrape)
	if !s.cfg.Metrics.Enabled {
		return nil
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	if s.cfg.Metrics.EnableGoCollector {
		reg.MustRegister(collectors.NewGoCollector())
	}
	if s.cfg.Metrics.EnableProcessCollector {
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	// Create Prometheus exporter (OTel Metric Reader)
	exp, err := otelprom.New(
		otelprom.WithRegisterer(reg),
	)
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
		sdkmetric.WithView(OTelMetricView()),
	)

	otel.SetMeterProvider(mp)

	s.engine.Use(OTelHTTPServerMetricsMiddleware())

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	s.engine.GET(s.cfg.Metrics.Path, gin.WrapH(h))

	// Register shutdown hook (important!)
	s.registerStopper(func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	return nil
}

func OTelMetricView() sdkmetric.View {
	return sdkmetric.NewView(
		sdkmetric.Instrument{
			Name: "http.server.*",
		},
		sdkmetric.Stream{
			AttributeFilter: func(kv attribute.KeyValue) bool {
				switch kv.Key {
				case "http.method", "http.route", "http.status_code":
					return true
				default:
					return false
				}
			},
		},
	)
}
func OTelHTTPServerMetricsMiddleware() gin.HandlerFunc {
	meter := otel.Meter("http.server")

	reqTotal, _ := meter.Int64Counter(
		"http.server.requests_total",
	)
	reqDur, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
	)
	inflight, _ := meter.Int64UpDownCounter(
		"http.server.active_requests",
	)

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		start := time.Now()

		inflight.Add(ctx, 1)
		defer inflight.Add(ctx, -1)

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		attrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", c.Writer.Status()),
		}

		reqTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
		reqDur.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	}
}
