package utils

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger provides structured logging functionality
type Logger struct {
	level  LogLevel
	output io.Writer
	prefix string
}

// NewLogger creates a new logger
func NewLogger(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stderr
	}
	
	return &Logger{
		level:  level,
		output: output,
		prefix: "[sb-yaml] ",
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger(verbose bool) *Logger {
	level := LogLevelInfo
	if verbose {
		level = LogLevelDebug
	}
	
	return NewLogger(level, os.Stderr)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetPrefix sets the log prefix
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, "DEBUG", format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, "INFO", format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, "WARN", format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, "ERROR", format, args...)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelError, "FATAL", format, args...)
	os.Exit(1)
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, levelStr, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s%s: %s\n", l.prefix, levelStr, message)
	
	fmt.Fprint(l.output, logLine)
}

// IsDebugEnabled returns whether debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= LogLevelDebug
}

// IsVerbose returns whether verbose logging is enabled (same as debug)
func (l *Logger) IsVerbose() bool {
	return l.IsDebugEnabled()
}

// WithPrefix returns a new logger with the specified prefix added
func (l *Logger) WithPrefix(prefix string) *Logger {
	newLogger := *l
	newLogger.prefix = l.prefix + prefix
	return &newLogger
}

// Quiet creates a logger that only logs errors
func (l *Logger) Quiet() *Logger {
	newLogger := *l
	newLogger.level = LogLevelError
	return &newLogger
}

// Silent creates a logger that doesn't log anything
func (l *Logger) Silent() *Logger {
	newLogger := *l
	newLogger.output = io.Discard
	return &newLogger
}

// Global logger instance
var defaultLogger = NewDefaultLogger(false)

// Global logging functions for convenience

// SetGlobalLevel sets the global logging level
func SetGlobalLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetGlobalVerbose sets global verbose logging
func SetGlobalVerbose(verbose bool) {
	if verbose {
		defaultLogger.SetLevel(LogLevelDebug)
	} else {
		defaultLogger.SetLevel(LogLevelInfo)
	}
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs an info message using the global logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the global logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs an error message and exits using the global logger
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return defaultLogger
}

// SetupStandardLogger configures the standard Go logger to use our format
func SetupStandardLogger(logger *Logger) {
	log.SetOutput(logger.output)
	log.SetPrefix(logger.prefix)
	log.SetFlags(0) // Remove timestamp since we handle our own formatting
}