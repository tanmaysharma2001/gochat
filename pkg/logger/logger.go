package logger

import (
	"log"
	"os"
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

func New() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.debugLogger.Printf(format, v...)
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
	os.Exit(1)
}

// Global logger instance
var GlobalLogger = New()

// Convenience functions
func Info(format string, v ...interface{}) {
	GlobalLogger.Info(format, v...)
}

func Error(format string, v ...interface{}) {
	GlobalLogger.Error(format, v...)
}

func Debug(format string, v ...interface{}) {
	GlobalLogger.Debug(format, v...)
}

func Fatal(format string, v ...interface{}) {
	GlobalLogger.Fatal(format, v...)
}