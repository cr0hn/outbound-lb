package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
	}{
		{"debug json", "debug", "json"},
		{"info json", "info", "json"},
		{"warn json", "warn", "json"},
		{"error json", "error", "json"},
		{"info text", "info", "text"},
		{"unknown level defaults to info", "unknown", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(tt.level, tt.format, &buf)
			if log == nil {
				t.Error("expected non-nil logger")
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	var buf bytes.Buffer
	log := New("debug", "text", &buf)

	// Replace default logger temporarily
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	Debug("debug message", "key", "value")
	if !strings.Contains(buf.String(), "debug message") {
		t.Error("expected debug message in output")
	}

	buf.Reset()
	Info("info message", "key", "value")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("expected info message in output")
	}

	buf.Reset()
	Warn("warn message", "key", "value")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("expected warn message in output")
	}

	buf.Reset()
	Error("error message", "key", "value")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("expected error message in output")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	log := New("info", "text", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	withLogger := With("component", "test")
	if withLogger == nil {
		t.Error("expected non-nil logger from With")
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	log := New("info", "text", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	groupLogger := WithGroup("test-group")
	if groupLogger == nil {
		t.Error("expected non-nil logger from WithGroup")
	}
}

func TestLogRequest(t *testing.T) {
	var buf bytes.Buffer
	log := New("info", "json", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	LogRequest("GET", "example.com", "127.0.0.1:1234", "192.168.1.1", 200, 100, 1024, 2048)

	output := buf.String()
	if !strings.Contains(output, "request") {
		t.Error("expected 'request' in output")
	}
	if !strings.Contains(output, "example.com") {
		t.Error("expected host in output")
	}
}

func TestLogBalancerSelection(t *testing.T) {
	var buf bytes.Buffer
	log := New("debug", "json", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	LogBalancerSelection("example.com", "192.168.1.1", 3)

	output := buf.String()
	if !strings.Contains(output, "balancer_selection") {
		t.Error("expected 'balancer_selection' in output")
	}
}

func TestLogConnectionLimit(t *testing.T) {
	var buf bytes.Buffer
	log := New("warn", "json", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	LogConnectionLimit("per_ip", "192.168.1.1", 100, 100)

	output := buf.String()
	if !strings.Contains(output, "connection_limit_reached") {
		t.Error("expected 'connection_limit_reached' in output")
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	log := New("error", "json", &buf)
	oldDefault := defaultLogger
	defaultLogger = log
	defer func() { defaultLogger = oldDefault }()

	LogError("test_operation", &testError{msg: "test error"}, "extra", "data")

	output := buf.String()
	if !strings.Contains(output, "test_operation") {
		t.Error("expected operation in output")
	}
	if !strings.Contains(output, "test error") {
		t.Error("expected error message in output")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestDefault(t *testing.T) {
	// Reset defaultLogger
	oldDefault := defaultLogger
	defaultLogger = nil
	defer func() { defaultLogger = oldDefault }()

	log := Default()
	if log == nil {
		t.Error("expected non-nil default logger")
	}
}
