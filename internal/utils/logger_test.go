package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelDebug, buf)
	
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	
	if logger.level != LogLevelDebug {
		t.Errorf("Logger level = %v, want %v", logger.level, LogLevelDebug)
	}
	
	if logger.output != buf {
		t.Error("Logger output not set correctly")
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		logFunc  func(*Logger, string, ...interface{})
		logLevel string
		message  string
		shouldLog bool
	}{
		{
			name:     "Debug at Debug level",
			level:    LogLevelDebug,
			logFunc:  (*Logger).Debug,
			logLevel: "DEBUG",
			message:  "debug message",
			shouldLog: true,
		},
		{
			name:     "Debug at Info level",
			level:    LogLevelInfo,
			logFunc:  (*Logger).Debug,
			logLevel: "DEBUG",
			message:  "debug message",
			shouldLog: false,
		},
		{
			name:     "Info at Info level",
			level:    LogLevelInfo,
			logFunc:  (*Logger).Info,
			logLevel: "INFO",
			message:  "info message",
			shouldLog: true,
		},
		{
			name:     "Warn at Info level",
			level:    LogLevelInfo,
			logFunc:  (*Logger).Warn,
			logLevel: "WARN",
			message:  "warning message",
			shouldLog: true,
		},
		{
			name:     "Error at Warn level",
			level:    LogLevelWarn,
			logFunc:  (*Logger).Error,
			logLevel: "ERROR",
			message:  "error message",
			shouldLog: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.level, buf)
			
			tt.logFunc(logger, tt.message)
			
			output := buf.String()
			if tt.shouldLog {
				if !strings.Contains(output, tt.logLevel) {
					t.Errorf("Expected log level %s not found in output", tt.logLevel)
				}
				if !strings.Contains(output, tt.message) {
					t.Errorf("Expected message '%s' not found in output", tt.message)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output, got: %s", output)
				}
			}
		})
	}
}

func TestLoggerFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelDebug, buf)
	
	logger.Info("Test %s with number %d", "formatting", 42)
	
	output := buf.String()
	expected := "Test formatting with number 42"
	
	if !strings.Contains(output, expected) {
		t.Errorf("Formatted message not found. Got: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelError, buf)
	
	// Should not log info
	logger.Info("info message")
	if buf.String() != "" {
		t.Error("Info message logged at Error level")
	}
	
	// Change level to Info
	buf.Reset()
	logger.SetLevel(LogLevelInfo)
	
	// Should now log info
	logger.Info("info message")
	if buf.String() == "" {
		t.Error("Info message not logged after level change")
	}
}

func TestSetPrefix(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelInfo, buf)
	
	customPrefix := "[CUSTOM] "
	logger.SetPrefix(customPrefix)
	
	logger.Info("test message")
	
	output := buf.String()
	if !strings.HasPrefix(output, customPrefix) {
		t.Errorf("Output doesn't start with custom prefix. Got: %s", output)
	}
}

func TestIsDebugEnabled(t *testing.T) {
	// Debug level
	logger := NewLogger(LogLevelDebug, nil)
	if !logger.IsDebugEnabled() {
		t.Error("IsDebugEnabled returned false for Debug level")
	}
	
	// Info level
	logger = NewLogger(LogLevelInfo, nil)
	if logger.IsDebugEnabled() {
		t.Error("IsDebugEnabled returned true for Info level")
	}
}

func TestIsVerbose(t *testing.T) {
	// Should be same as IsDebugEnabled
	logger := NewLogger(LogLevelDebug, nil)
	if !logger.IsVerbose() {
		t.Error("IsVerbose returned false for Debug level")
	}
	
	logger = NewLogger(LogLevelInfo, nil)
	if logger.IsVerbose() {
		t.Error("IsVerbose returned true for Info level")
	}
}

func TestWithPrefix(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelInfo, buf)
	logger.SetPrefix("[MAIN] ")
	
	subLogger := logger.WithPrefix("[SUB] ")
	
	subLogger.Info("test")
	
	output := buf.String()
	if !strings.Contains(output, "[MAIN] [SUB]") {
		t.Errorf("Prefixes not combined correctly. Got: %s", output)
	}
}

func TestQuiet(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelInfo, buf)
	
	quietLogger := logger.Quiet()
	
	// Should not log info
	quietLogger.Info("info")
	if buf.String() != "" {
		t.Error("Quiet logger logged info message")
	}
	
	// Should log error
	buf.Reset()
	quietLogger.Error("error")
	if buf.String() == "" {
		t.Error("Quiet logger didn't log error message")
	}
}

func TestSilent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelDebug, buf)
	
	silentLogger := logger.Silent()
	
	// Should not log anything
	silentLogger.Debug("debug")
	silentLogger.Info("info")
	silentLogger.Warn("warn")
	silentLogger.Error("error")
	
	if buf.String() != "" {
		t.Error("Silent logger produced output")
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Save original logger
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()
	
	buf := &bytes.Buffer{}
	defaultLogger = NewLogger(LogLevelInfo, buf)
	
	// Test global functions
	Info("global info")
	if !strings.Contains(buf.String(), "global info") {
		t.Error("Global Info function didn't log")
	}
	
	buf.Reset()
	Warn("global warn")
	if !strings.Contains(buf.String(), "global warn") {
		t.Error("Global Warn function didn't log")
	}
	
	buf.Reset()
	Error("global error")
	if !strings.Contains(buf.String(), "global error") {
		t.Error("Global Error function didn't log")
	}
	
	// Debug should not log at Info level
	buf.Reset()
	Debug("global debug")
	if buf.String() != "" {
		t.Error("Global Debug function logged at Info level")
	}
}

func TestSetGlobalVerbose(t *testing.T) {
	// Save original logger
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()
	
	buf := &bytes.Buffer{}
	defaultLogger = NewLogger(LogLevelInfo, buf)
	
	// Enable verbose
	SetGlobalVerbose(true)
	
	Debug("debug after verbose")
	if !strings.Contains(buf.String(), "debug after verbose") {
		t.Error("Debug not logged after SetGlobalVerbose(true)")
	}
	
	// Disable verbose
	buf.Reset()
	SetGlobalVerbose(false)
	
	Debug("debug after non-verbose")
	if buf.String() != "" {
		t.Error("Debug logged after SetGlobalVerbose(false)")
	}
}

func TestNewDefaultLogger(t *testing.T) {
	// Non-verbose
	logger := NewDefaultLogger(false)
	if logger.level != LogLevelInfo {
		t.Errorf("Default logger level = %v, want %v", logger.level, LogLevelInfo)
	}
	
	// Verbose
	logger = NewDefaultLogger(true)
	if logger.level != LogLevelDebug {
		t.Errorf("Verbose logger level = %v, want %v", logger.level, LogLevelDebug)
	}
}