package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

// Fields represents structured logging fields
type Fields map[string]interface{}

// Logger interface defines logging operations
type Logger interface {
	Debug(msg string, fields ...Fields)
	Info(msg string, fields ...Fields)
	Warn(msg string, fields ...Fields)
	Error(msg string, fields ...Fields)
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

// DefaultLogger implements the Logger interface
type DefaultLogger struct {
	level  LogLevel
	output io.Writer
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel, output io.Writer) Logger {
	if output == nil {
		output = os.Stdout
	}
	
	return &DefaultLogger{
		level:  level,
		output: output,
		logger: log.New(output, "", 0), // No default prefix or flags
	}
}

// NewLoggerFromConfig creates a logger from configuration
func NewLoggerFromConfig(levelStr, filename string) (Logger, error) {
	level, err := ParseLogLevel(levelStr)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	var output io.Writer = os.Stdout
	if filename != "" {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", filename, err)
		}
		output = file
	}

	return NewLogger(level, output), nil
}

// Debug logs a debug message with optional fields
func (l *DefaultLogger) Debug(msg string, fields ...Fields) {
	if l.level <= DebugLevel {
		l.log(DebugLevel, msg, fields...)
	}
}

// Info logs an info message with optional fields
func (l *DefaultLogger) Info(msg string, fields ...Fields) {
	if l.level <= InfoLevel {
		l.log(InfoLevel, msg, fields...)
	}
}

// Warn logs a warning message with optional fields
func (l *DefaultLogger) Warn(msg string, fields ...Fields) {
	if l.level <= WarnLevel {
		l.log(WarnLevel, msg, fields...)
	}
}

// Error logs an error message with optional fields
func (l *DefaultLogger) Error(msg string, fields ...Fields) {
	if l.level <= ErrorLevel {
		l.log(ErrorLevel, msg, fields...)
	}
}

// SetLevel sets the minimum log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level
func (l *DefaultLogger) GetLevel() LogLevel {
	return l.level
}

// log formats and writes a log message
func (l *DefaultLogger) log(level LogLevel, msg string, fields ...Fields) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// Build the log message
	logMsg := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), msg)
	
	// Add structured fields if provided
	if len(fields) > 0 && fields[0] != nil {
		var fieldStrs []string
		for key, value := range fields[0] {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", key, value))
		}
		if len(fieldStrs) > 0 {
			logMsg += " | " + strings.Join(fieldStrs, " ")
		}
	}
	
	l.logger.Println(logMsg)
}

// RequestLogger creates a logger with request-specific fields
func (l *DefaultLogger) RequestLogger(requestID, method, path, remoteAddr string) Logger {
	return &RequestLogger{
		parent: l,
		fields: Fields{
			"request_id":  requestID,
			"method":      method,
			"path":        path,
			"remote_addr": remoteAddr,
		},
	}
}

// RequestLogger wraps a logger with request-specific context
type RequestLogger struct {
	parent Logger
	fields Fields
}

// Debug logs a debug message with request context
func (rl *RequestLogger) Debug(msg string, fields ...Fields) {
	mergedFields := rl.mergeFields(fields...)
	rl.parent.Debug(msg, mergedFields)
}

// Info logs an info message with request context
func (rl *RequestLogger) Info(msg string, fields ...Fields) {
	mergedFields := rl.mergeFields(fields...)
	rl.parent.Info(msg, mergedFields)
}

// Warn logs a warning message with request context
func (rl *RequestLogger) Warn(msg string, fields ...Fields) {
	mergedFields := rl.mergeFields(fields...)
	rl.parent.Warn(msg, mergedFields)
}

// Error logs an error message with request context
func (rl *RequestLogger) Error(msg string, fields ...Fields) {
	mergedFields := rl.mergeFields(fields...)
	rl.parent.Error(msg, mergedFields)
}

// SetLevel sets the minimum log level on the parent logger
func (rl *RequestLogger) SetLevel(level LogLevel) {
	rl.parent.SetLevel(level)
}

// GetLevel returns the current log level from the parent logger
func (rl *RequestLogger) GetLevel() LogLevel {
	return rl.parent.GetLevel()
}

// mergeFields combines request fields with additional fields
func (rl *RequestLogger) mergeFields(fields ...Fields) Fields {
	merged := make(Fields)
	
	// Copy request fields
	for k, v := range rl.fields {
		merged[k] = v
	}
	
	// Add additional fields
	if len(fields) > 0 && fields[0] != nil {
		for k, v := range fields[0] {
			merged[k] = v
		}
	}
	
	return merged
}