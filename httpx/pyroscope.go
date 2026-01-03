package httpx

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/grafana/pyroscope-go"
)

func PyroscopeGinLabels() gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		pyroscope.TagWrapper(
			c.Request.Context(),
			pyroscope.Labels(
				"http.method", c.Request.Method,
				"http.route", route,
			),
			func(ctx context.Context) {
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			},
		)
	}
}

func (s *Server) initProfiling() error {
	p := s.cfg.Profiling
	if !p.Enabled {
		return nil
	}

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: s.cfg.Name,
		ServerAddress:   p.ServerAddress,
		Tags:            p.Tags,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return fmt.Errorf("pyroscope start: %w", err)
	}

	s.registerStopper(func() error {
		return profiler.Stop()
	})

	if p.TagByRoute {
		s.engine.Use(PyroscopeGinLabels())
	}
	return nil
}
