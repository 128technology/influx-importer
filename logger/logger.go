package logger

import (
	"log"
	"os"
)

// Log represents singleton access to the default logger
var Log = NewLogger("")

// Logger represents a logger
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new Logger
func NewLogger(prefix string) Logger {
	return Logger{
		logger: log.New(os.Stdout, prefix, log.LUTC|log.Ldate|log.Ltime|log.Lmicroseconds),
	}
}

func (l Logger) print(level string, format string, args ...interface{}) {
	l.logger.Printf(level+" "+format, args...)
}

func (l Logger) Debug(format string, args ...interface{}) {
	l.print("DEBG", format, args...)
}

func (l Logger) Info(format string, args ...interface{}) {
	l.print("INFO", format, args...)
}

func (l Logger) Warn(format string, args ...interface{}) {
	l.print("WARN", format, args...)
}

func (l Logger) Error(format string, args ...interface{}) {
	l.print("ERRO", format, args...)
}

func (l Logger) Fatal(format string, args ...interface{}) {
	l.logger.Fatalf("FATL "+format, args...)
}
