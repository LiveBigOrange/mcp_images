package vlm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_images/internal/config"
	"mcp_images/internal/logger"
)

func TestBuildRequest(t *testing.T) {
	cfg := &config.Config{Model: "test-model", APIBase: "https://api.example.com/v1/chat/completions"}
	req := BuildRequest(cfg, "data:image/jpeg;base64,abc123")

	if req.Model != "test-model" {
		t.Errorf("Model = %q, want test-model", req.Model)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("First message role = %q, want system", req.Messages[0].Role)
	}
	if req.Messages[1].Role != "user" {
		t.Errorf("Second message role = %q, want user", req.Messages[1].Role)
	}
	if req.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", req.MaxTokens)
	}
}

func TestParseResponse(t *testing.T) {
	resp := VLMResponse{
		Choices: []VLMChoice{
			{Message: struct {
				Content string `json:"content"`
			}{Content: "test analysis result"}},
		},
	}
	body, _ := json.Marshal(resp)

	result, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if result != "test analysis result" {
		t.Errorf("result = %q, want test analysis result", result)
	}
}

func TestParseResponse_EmptyChoices(t *testing.T) {
	resp := VLMResponse{Choices: []VLMChoice{}}
	body, _ := json.Marshal(resp)

	_, err := ParseResponse(body)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestParseResponse_EmptyContent(t *testing.T) {
	resp := VLMResponse{
		Choices: []VLMChoice{
			{Message: struct {
				Content string `json:"content"`
			}{Content: ""}},
		},
	}
	body, _ := json.Marshal(resp)

	_, err := ParseResponse(body)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		statusCode int
		contains   string
	}{
		{401, "认证失败"},
		{403, "认证失败"},
		{404, "模型不存在"},
		{429, "频率超限"},
		{500, "内部错误"},
		{200, "未知错误"},
	}
	for _, tt := range tests {
		err := MapError(nil, tt.statusCode)
		if err == nil {
			t.Errorf("MapError(%d) returned nil", tt.statusCode)
			continue
		}
	}
}

func TestAnalyzeImage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}
		resp := VLMResponse{
			Choices: []VLMChoice{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "analysis result"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		APIBase: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	}
	lg := logger.New("error")
	client := NewClient(cfg, lg)

	result, err := client.AnalyzeImage(context.Background(), "data:image/jpeg;base64,abc")
	if err != nil {
		t.Fatalf("AnalyzeImage failed: %v", err)
	}
	if result != "analysis result" {
		t.Errorf("result = %q, want analysis result", result)
	}
}

func TestAnalyzeImage_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cfg := &config.Config{
		APIBase: server.URL,
		APIKey:  "wrong-key",
		Model:   "test-model",
	}
	lg := logger.New("error")
	client := NewClient(cfg, lg)

	_, err := client.AnalyzeImage(context.Background(), "data:image/jpeg;base64,abc")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}
