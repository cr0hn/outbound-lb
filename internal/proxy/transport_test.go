package proxy

import (
	"testing"
	"time"
)

func TestNewTransportPool(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2"}
	tp := NewTransportPool(ips, 30*time.Second)

	if tp == nil {
		t.Fatal("expected non-nil transport pool")
	}

	if len(tp.transports) != 2 {
		t.Errorf("expected 2 transports, got %d", len(tp.transports))
	}
}

func TestTransportPool_Get(t *testing.T) {
	ips := []string{"127.0.0.1"}
	tp := NewTransportPool(ips, 30*time.Second)

	// Get existing transport
	tr := tp.Get("127.0.0.1")
	if tr == nil {
		t.Error("expected non-nil transport for existing IP")
	}

	// Get same transport again
	tr2 := tp.Get("127.0.0.1")
	if tr != tr2 {
		t.Error("expected same transport instance for same IP")
	}

	// Get transport for new IP (should create one)
	tr3 := tp.Get("127.0.0.3")
	if tr3 == nil {
		t.Error("expected non-nil transport for new IP")
	}
}

func TestTransportPool_Close(t *testing.T) {
	ips := []string{"127.0.0.1"}
	tp := NewTransportPool(ips, 30*time.Second)

	// Should not panic
	tp.Close()
}

func TestNewDialer(t *testing.T) {
	d := NewDialer("127.0.0.1", 30*time.Second, 60*time.Second)

	if d == nil {
		t.Fatal("expected non-nil dialer")
	}

	if d.localIP != "127.0.0.1" {
		t.Errorf("expected localIP 127.0.0.1, got %s", d.localIP)
	}

	if d.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", d.timeout)
	}

	if d.idleTimeout != 60*time.Second {
		t.Errorf("expected idleTimeout 60s, got %v", d.idleTimeout)
	}
}
