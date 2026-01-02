package httpx

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestServer_StartAndStop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := New(
		WithName("test-api"),
		WithAddr(":18081"),
		WithRoutes(func(r *gin.Engine) {
			r.GET("/healthz", func(c *gin.Context) {
				c.String(200, "ok")
			})
		}),
	)

	// ---- Start server ----
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Wait until the server is reachable
	if err := waitForTCP("127.0.0.1:18081", time.Second); err != nil {
		t.Fatalf("server did not start listening: %v", err)
	}

	// Verify HTTP works
	resp, err := http.Get("http://127.0.0.1:18081/healthz")
	if err != nil {
		t.Fatalf("http call failed: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// ---- Stop server ----
	if err := srv.Stop(); err != nil {
		t.Fatalf("failed to stop server: %v", err)
	}

	// ---- Verify server is down ----
	_, err = http.Get("http://127.0.0.1:18081/healthz")
	if err == nil {
		t.Fatal("expected error after server stopped, got nil")
	}
}

func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return context.DeadlineExceeded
}

func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
