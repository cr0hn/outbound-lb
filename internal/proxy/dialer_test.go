package proxy

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDialer_Dial(t *testing.T) {
	// Start a test server to dial
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	d := NewDialer("127.0.0.1", 5*time.Second, 10*time.Second)

	conn, err := d.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	conn.Close()
}

func TestDialer_DialContext(t *testing.T) {
	// Start a test server to dial
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	d := NewDialer("127.0.0.1", 5*time.Second, 10*time.Second)

	ctx := context.Background()
	conn, err := d.DialContext(ctx, "tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial context failed: %v", err)
	}
	conn.Close()
}

func TestDialer_DialContext_Timeout(t *testing.T) {
	d := NewDialer("127.0.0.1", 100*time.Millisecond, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Try to dial a non-routable IP to trigger timeout
	_, err := d.DialContext(ctx, "tcp", "10.255.255.1:12345")
	if err == nil {
		t.Error("expected timeout error")
	}
}
