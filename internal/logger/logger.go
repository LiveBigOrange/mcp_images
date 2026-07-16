package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Field struct {
	Key   string
	Value interface{}
}

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

type StderrLogger struct {
	mu     sync.Mutex
	level  Level
	writer io.Writer
}

func New(level string) *StderrLogger {
	l := &StderrLogger{
		writer: os.Stderr,
		level:  parseLevel(level),
	}
	return l
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelWarn
	}
}

func (l *StderrLogger) log(level Level, msg string, fields ...Field) {
	if level < l.level {
		return
	}
	var levelStr string
	switch level {
	case LevelDebug:
		levelStr = "DEBUG"
	case LevelInfo:
		levelStr = "INFO"
	case LevelWarn:
		levelStr = "WARN"
	case LevelError:
		levelStr = "ERROR"
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	line := fmt.Sprintf("%s [%s] %s", timestamp, levelStr, msg)

	for _, f := range fields {
		line += fmt.Sprintf(" %s=%v", f.Key, f.Value)
	}

	l.mu.Lock()
	fmt.Fprintln(l.writer, line)
	l.mu.Unlock()
}

func (l *StderrLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

func (l *StderrLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

func (l *StderrLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

func (l *StderrLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 7 {
		return "****"
	}
	return key[:3] + "****" + key[len(key)-4:]
}
