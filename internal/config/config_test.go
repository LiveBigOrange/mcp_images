package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("VLM_API_BASE", " https://api.example.com/v1/chat/completions ")
	t.Setenv("VLM_API_KEY", " sk-test-key ")
	t.Setenv("VLM_MODEL", " qwen2.5-v:2b ")
	t.Setenv("VLM_LOG_LEVEL", " debug ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.APIBase != "https://api.example.com/v1/chat/completions" {
		t.Errorf("APIBase trim failed: got %q", cfg.APIBase)
	}
	if cfg.APIKey != "sk-test-key" {
		t.Errorf("APIKey trim failed: got %q", cfg.APIKey)
	}
	if cfg.Model != "qwen2.5-v:2b" {
		t.Errorf("Model trim failed: got %q", cfg.Model)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel trim failed: got %q", cfg.LogLevel)
	}
}

func TestValidate_MissingAPIBase(t *testing.T) {
	cfg := &Config{APIBase: "", Model: "test"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing APIBase")
	}
}

func TestValidate_MissingModel(t *testing.T) {
	cfg := &Config{APIBase: "https://api.example.com", Model: ""}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing Model")
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := &Config{APIBase: "ftp://example.com", Model: "test"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestValidate_CloudMissingKey(t *testing.T) {
	cfg := &Config{APIBase: "https://api.example.com", Model: "test", APIKey: ""}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for cloud API without key")
	}
}

func TestValidate_LocalNoKey(t *testing.T) {
	cfg := &Config{APIBase: "http://localhost:11434/v1/chat/completions", Model: "test", APIKey: ""}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("local API should not require key: %v", err)
	}
}

func TestNormalizeAPIBase(t *testing.T) {
	cfg := &Config{APIBase: "https://api.example.com/"}
	cfg.NormalizeAPIBase()
	if cfg.APIBase != "https://api.example.com" {
		t.Errorf("trailing slash not removed: got %q", cfg.APIBase)
	}
}

func TestValidateLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"INFO", "info"},
		{"Warn", "warn"},
		{"ERROR", "error"},
		{"", "warn"},
		{"invalid", "warn"},
	}
	for _, tt := range tests {
		cfg := &Config{LogLevel: tt.input}
		cfg.ValidateLogLevel()
		if cfg.LogLevel != tt.expected {
			t.Errorf("LogLevel %q: expected %q, got %q", tt.input, tt.expected, cfg.LogLevel)
		}
	}
}

func TestIsLocalAPI(t *testing.T) {
	tests := []struct {
		apiBase  string
		expected bool
	}{
		{"http://localhost:11434", true},
		{"http://127.0.0.1:11434", true},
		{"https://api.example.com", false},
	}
	for _, tt := range tests {
		cfg := &Config{APIBase: tt.apiBase}
		if got := cfg.IsLocalAPI(); got != tt.expected {
			t.Errorf("IsLocalAPI(%q) = %v, want %v", tt.apiBase, got, tt.expected)
		}
	}
}
