package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging for the IRC server
type Logger struct {
	logger *log.Logger
	level  LogLevel
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
		level:  INFO,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) log(level LogLevel, levelStr string, msg string, keysAndValues ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	output := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, msg)

	// Add key-value pairs
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			output += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}

	l.logger.Println(output)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DEBUG, "DEBUG", msg, keysAndValues...)
}

// Info logs an info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.log(INFO, "INFO", msg, keysAndValues...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WARN, "WARN", msg, keysAndValues...)
}

// Error logs an error message
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.log(ERROR, "ERROR", msg, keysAndValues...)
}
