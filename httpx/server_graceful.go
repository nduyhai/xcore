package httpx

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// RunGraceful starts the server and blocks until SIGINT/SIGTERM,
// then gracefully stops ite using Server.Stop().
func (s *Server) RunGraceful() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return s.RunGracefulContext(ctx)
}

// RunGracefulContext starts the server and blocks until ctx is done,
// then gracefully stops it using Server.Stop().
func (s *Server) RunGracefulContext(ctx context.Context) error {
	if err := s.Start(); err != nil {
		return err
	}

	<-ctx.Done() // wait for shutdown signal/cancel

	// use background here because ctx is already canceled
	return s.Stop()
}
