package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
		hasError bool
	}{
		{"debug", DebugLevel, false},
		{"DEBUG", DebugLevel, false},
		{"info", InfoLevel, false},
		{"INFO", InfoLevel, false},
		{"warn", WarnLevel, false},
		{"warning", WarnLevel, false},
		{"WARN", WarnLevel, false},
		{"error", ErrorLevel, false},
		{"ERROR", ErrorLevel, false},
		{"invalid", InfoLevel, true},
		{"", InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLogLevel(tt.input)
			if tt.hasError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if level != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, level)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.GetLevel() != InfoLevel {
		t.Errorf("Expected level %v, got %v", InfoLevel, logger.GetLevel())
	}
}

func TestNewLogger_NilOutput(t *testing.T) {
	logger := NewLogger(InfoLevel, nil)
	if logger == nil {
		t.Fatal("Expected logger to be created even with nil output")
	}
}

func TestNewLoggerFromConfig(t *testing.T) {
	// Test with stdout
	logger, err := NewLoggerFromConfig("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	if logger.GetLevel() != InfoLevel {
		t.Errorf("Expected level %v, got %v", InfoLevel, logger.GetLevel())
	}

	// Test with file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger, err = NewLoggerFromConfig("debug", logFile)
	if err != nil {
		t.Fatalf("Failed to create logger with file: %v", err)
	}
	if logger.GetLevel() != DebugLevel {
		t.Errorf("Expected level %v, got %v", DebugLevel, logger.GetLevel())
	}

	// Test with invalid level
	_, err = NewLoggerFromConfig("invalid", "")
	if err == nil {
		t.Error("Expected error for invalid log level")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Debug should not be logged (below threshold)
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged when level is Info")
	}

	// Info should be logged
	logger.Info("info message")
	output := buf.String()
	if !strings.Contains(output, "INFO: info message") {
		t.Errorf("Expected info message in output, got: %s", output)
	}

	buf.Reset()

	// Warn should be logged
	logger.Warn("warn message")
	output = buf.String()
	if !strings.Contains(output, "WARN: warn message") {
		t.Errorf("Expected warn message in output, got: %s", output)
	}

	buf.Reset()

	// Error should be logged
	logger.Error("error message")
	output = buf.String()
	if !strings.Contains(output, "ERROR: error message") {
		t.Errorf("Expected error message in output, got: %s", output)
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	fields := Fields{
		"user_id": 123,
		"action":  "login",
	}

	logger.Info("user action", fields)
	output := buf.String()

	if !strings.Contains(output, "INFO: user action") {
		t.Errorf("Expected log message in output, got: %s", output)
	}
	if !strings.Contains(output, "user_id=123") {
		t.Errorf("Expected user_id field in output, got: %s", output)
	}
	if !strings.Contains(output, "action=login") {
		t.Errorf("Expected action field in output, got: %s", output)
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Debug should not be logged initially
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged when level is Info")
	}

	// Change level to Debug
	logger.SetLevel(DebugLevel)
	if logger.GetLevel() != DebugLevel {
		t.Errorf("Expected level %v, got %v", DebugLevel, logger.GetLevel())
	}

	// Now debug should be logged
	logger.Debug("debug message")
	output := buf.String()
	if !strings.Contains(output, "DEBUG: debug message") {
		t.Errorf("Expected debug message in output, got: %s", output)
	}
}

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	parentLogger := NewLogger(InfoLevel, &buf).(*DefaultLogger)

	requestLogger := parentLogger.RequestLogger("req-123", "GET", "/api/test", "192.168.1.1")

	requestLogger.Info("processing request")
	output := buf.String()

	if !strings.Contains(output, "INFO: processing request") {
		t.Errorf("Expected log message in output, got: %s", output)
	}
	if !strings.Contains(output, "request_id=req-123") {
		t.Errorf("Expected request_id field in output, got: %s", output)
	}
	if !strings.Contains(output, "method=GET") {
		t.Errorf("Expected method field in output, got: %s", output)
	}
	if !strings.Contains(output, "path=/api/test") {
		t.Errorf("Expected path field in output, got: %s", output)
	}
	if !strings.Contains(output, "remote_addr=192.168.1.1") {
		t.Errorf("Expected remote_addr field in output, got: %s", output)
	}
}

func TestRequestLogger_WithAdditionalFields(t *testing.T) {
	var buf bytes.Buffer
	parentLogger := NewLogger(InfoLevel, &buf).(*DefaultLogger)

	requestLogger := parentLogger.RequestLogger("req-456", "POST", "/api/users", "10.0.0.1")

	additionalFields := Fields{
		"user_id": 789,
		"status":  "success",
	}

	requestLogger.Info("request completed", additionalFields)
	output := buf.String()

	// Should contain both request context and additional fields
	if !strings.Contains(output, "request_id=req-456") {
		t.Errorf("Expected request_id field in output, got: %s", output)
	}
	if !strings.Contains(output, "user_id=789") {
		t.Errorf("Expected user_id field in output, got: %s", output)
	}
	if !strings.Contains(output, "status=success") {
		t.Errorf("Expected status field in output, got: %s", output)
	}
}

func TestRequestLogger_LevelOperations(t *testing.T) {
	var buf bytes.Buffer
	parentLogger := NewLogger(InfoLevel, &buf).(*DefaultLogger)
	requestLogger := parentLogger.RequestLogger("req-789", "GET", "/test", "127.0.0.1")

	// Test getting level
	if requestLogger.GetLevel() != InfoLevel {
		t.Errorf("Expected level %v, got %v", InfoLevel, requestLogger.GetLevel())
	}

	// Test setting level
	requestLogger.SetLevel(DebugLevel)
	if requestLogger.GetLevel() != DebugLevel {
		t.Errorf("Expected level %v after setting, got %v", DebugLevel, requestLogger.GetLevel())
	}

	// Verify parent logger level was also changed
	if parentLogger.GetLevel() != DebugLevel {
		t.Errorf("Expected parent level %v, got %v", DebugLevel, parentLogger.GetLevel())
	}
}