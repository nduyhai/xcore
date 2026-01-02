package httpx

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestServer_RunGracefulContext(t *testing.T) {
	// Gin test mode (important for tests)
	gin.SetMode(gin.TestMode)

	// Use a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create server
	srv := New(
		WithName("test-api"),
		WithAddr(":18080"), // fixed port for test
		WithRoutes(func(r *gin.Engine) {
			r.GET("/healthz", func(c *gin.Context) {
				c.String(200, "ok")
			})
		}),
	)

	// Run server in the background
	done := make(chan error, 1)
	go func() {
		done <- srv.RunGracefulContext(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// ---- Verify server responds ----
	resp, err := http.Get("http://127.0.0.1:18080/healthz")
	if err != nil {
		t.Fatalf("failed to call server: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// ---- Trigger graceful shutdown ----
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shutdown gracefully in time")
	}
}
