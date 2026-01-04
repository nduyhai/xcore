package httpx

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func (s *Server) initObservability() error {
	switch s.cfg.Obs.Mode {
	case ObsModeAutoExport:
		return s.initObservabilityAutoExport()
	case ObsModeManual, "":
		return s.initObservabilityManual()
	default:
		return fmt.Errorf("unknown obs mode: %s", s.cfg.Obs.Mode)
	}
}

func (s *Server) initObservabilityAutoExport() error {

	if !s.cfg.Obs.Enabled {
		return nil
	}
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	exp, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return err
	}

	var tpOpts []sdktrace.TracerProviderOption
	if !autoexport.IsNoneSpanExporter(exp) {
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)
	s.registerStopper(func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return err
	}

	var mpOpts []metric.Option
	if !autoexport.IsNoneMetricReader(reader) {
		mpOpts = append(mpOpts, metric.WithReader(reader))
	}
	mp := metric.NewMeterProvider(mpOpts...)
	otel.SetMeterProvider(mp)

	s.registerStopper(func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	return s.initOTelHttp()
}

func (s *Server) initObservabilityManual() error {
	var errs []error

	if err := s.initTracerProvider(); err != nil {
		errs = append(errs, err)
	}

	if err := s.initMetrics(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)

}

func (s *Server) initOTelHttp() error {
	s.engine.Use(otelgin.Middleware(s.cfg.Name))
	return nil
}
