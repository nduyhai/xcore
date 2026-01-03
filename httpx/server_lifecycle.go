package httpx

import (
	"context"
	"errors"
	"net/http"
)

func (s *Server) Start() error {
	go func() {
		s.log.Info("http server starting", "name", s.cfg.Name, "addr", s.cfg.Addr)
		if err := s.httpS.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server failed", "err", err)
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	stopCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	s.log.Info("http server stopping", "name", s.cfg.Name, "timeout", s.cfg.ShutdownTimeout.String())

	var errs []error

	if s.httpS != nil {
		if err := s.httpS.Shutdown(stopCtx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := s.stopAll(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
