package main

import (
	"fmt"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var logLevel = INFO

// SetLogLevel sets the minimum log level to display
func SetLogLevel(level LogLevel) {
	logLevel = level
}

// Log prints a message with timestamp and level if it meets the minimum level
func Log(level LogLevel, format string, args ...interface{}) {
	if level < logLevel {
		return
	}
	
	levelStr := "INFO"
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case WARN:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	}
	
	timestamp := time.Now().Format("2025-03-09 15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", timestamp, levelStr, message)
} 