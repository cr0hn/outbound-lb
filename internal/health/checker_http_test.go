package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 5*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err != nil {
		t.Errorf("expected check to succeed, got error: %v", err)
	}
}

func TestHTTPChecker_Check_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 5*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	// 3xx should be considered success
	if err != nil {
		t.Errorf("expected 301 to be success, got error: %v", err)
	}
}

func TestHTTPChecker_Check_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 5*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected 500 to fail check")
	}
}

func TestHTTPChecker_Check_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 5*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected 404 to fail check")
	}
}

func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) {
	// Use a URL that won't connect
	checker := NewHTTPChecker("http://127.0.0.1:59999/health", 1*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected connection refused to fail check")
	}
}

func TestHTTPChecker_Check_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than timeout
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 100*time.Millisecond)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected timeout to fail check")
	}
}

func TestHTTPChecker_Check_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.URL, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected context cancellation to fail check")
	}
}

func TestHTTPChecker_Check_InvalidURL(t *testing.T) {
	checker := NewHTTPChecker("://invalid-url", 1*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected invalid URL to fail check")
	}
}
