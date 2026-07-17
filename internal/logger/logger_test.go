package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer
	lg := &StderrLogger{level: LevelWarn, writer: &buf}

	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Warn("warn msg")
	lg.Error("error msg")

	output := buf.String()
	if strings.Contains(output, "debug msg") {
		t.Error("debug should be filtered at warn level")
	}
	if strings.Contains(output, "info msg") {
		t.Error("info should be filtered at warn level")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("warn should be present")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("error should be present")
	}
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	lg := &StderrLogger{level: LevelDebug, writer: &buf}
	lg.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("log should contain [INFO]")
	}
	if !strings.Contains(output, "test message") {
		t.Error("log should contain message")
	}
}

func TestNew(t *testing.T) {
	lg := New("debug")
	if lg.level != LevelDebug {
		t.Errorf("expected LevelDebug, got %v", lg.level)
	}
	lg = New("invalid")
	if lg.level != LevelWarn {
		t.Errorf("expected LevelWarn for invalid, got %v", lg.level)
	}
}
