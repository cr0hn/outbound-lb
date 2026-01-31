// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// requestIDKey is the context key for request IDs.
type requestIDKey struct{}

// requestCounter provides a monotonically increasing counter for request IDs.
var requestCounter atomic.Uint64

// GenerateRequestID generates a unique request ID.
// Format: timestamp_nanos-counter-random8bytes
// This ensures uniqueness even across restarts and high concurrency.
func GenerateRequestID() string {
	counter := requestCounter.Add(1)
	timestamp := time.Now().UnixNano()

	// Generate 4 random bytes for additional entropy
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	return fmt.Sprintf("%d-%d-%s", timestamp, counter, hex.EncodeToString(randomBytes))
}

// ContextWithRequestID returns a new context with the request ID attached.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDFromContext extracts the request ID from the context.
// Returns empty string if no request ID is present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
