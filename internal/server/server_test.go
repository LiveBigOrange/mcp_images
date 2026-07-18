package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"mcp_images/internal/config"
	"mcp_images/internal/logger"
)

func newTestServer(t *testing.T) (*MCPServer, *bytes.Buffer) {
	t.Helper()
	cfg := &config.Config{APIBase: "http://localhost:11434/v1/chat/completions", Model: "m", MaxDimension: 2048}
	s := NewServer(cfg, logger.New("error"))
	buf := &bytes.Buffer{}
	s.stdoutW = bufio.NewWriter(buf)
	return s, buf
}

func TestWriteResponse_CapturesJSON(t *testing.T) {
	s, buf := newTestServer(t)
	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(7),
		Result:  map[string]interface{}{"ok": true},
	})
	out := buf.Bytes()
	if !strings.HasPrefix(string(out), "{\"jsonrpc\":\"2.0\"") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.HasSuffix(string(out), "\n") {
		t.Error("response should be newline-terminated")
	}
	var got JSONRPCResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q", got.JSONRPC)
	}
}

func TestWriteResponse_SkipsEmpty(t *testing.T) {
	s, buf := newTestServer(t)
	s.writeResponse(JSONRPCResponse{})
	if buf.Len() != 0 {
		t.Errorf("empty response should not be written, got %q", buf.Bytes())
	}
}

func TestHandlePing(t *testing.T) {
	s, _ := newTestServer(t)
	resp := s.handlePing(JSONRPCRequest{ID: float64(1)})
	if resp.ID != float64(1) {
		t.Errorf("ID = %v", resp.ID)
	}
	if _, ok := resp.Result.(map[string]interface{}); !ok {
		t.Error("ping result should be an empty object")
	}
}

func TestHandleInitialize(t *testing.T) {
	s, _ := newTestServer(t)
	resp := s.handleInitialize(JSONRPCRequest{ID: float64(1)})
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T", resp.Result)
	}
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
	info, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("serverInfo missing")
	}
	if info["name"] != "mcp_images" {
		t.Errorf("name = %v", info["name"])
	}
}

func TestNotInitializedResp(t *testing.T) {
	resp := notInitializedResp(float64(42))
	if resp.Error == nil || resp.Error.Code != -32002 {
		t.Errorf("expected -32002 error, got %+v", resp.Error)
	}
}

func TestIsNotification(t *testing.T) {
	s, _ := newTestServer(t)
	if !s.isNotification(JSONRPCRequest{ID: nil}) {
		t.Error("req with nil ID should be a notification")
	}
	if s.isNotification(JSONRPCRequest{ID: float64(1)}) {
		t.Error("req with non-nil ID should not be a notification")
	}
}

func TestHandleToolsCall_NotInitialized(t *testing.T) {
	s, _ := newTestServer(t)
	resp := s.handleRequest(nil, JSONRPCRequest{ID: float64(1), Method: "tools/call", Params: map[string]interface{}{"name": "", "arguments": map[string]interface{}{}}})
	if resp.Error == nil || resp.Error.Code != -32002 {
		t.Errorf("expected -32002 not-initialized, got %+v", resp.Error)
	}
}

func TestHandleToolsList_NotInitialized(t *testing.T) {
	s, _ := newTestServer(t)
	resp := s.handleRequest(nil, JSONRPCRequest{ID: float64(1), Method: "tools/list"})
	if resp.Error == nil || resp.Error.Code != -32002 {
		t.Errorf("expected -32002 not-initialized, got %+v", resp.Error)
	}
}

func TestHandleToolsList_AfterInit(t *testing.T) {
	s, _ := newTestServer(t)
	s.initMu.Lock()
	s.initialized = true
	s.initMu.Unlock()
	resp := s.handleToolsList(JSONRPCRequest{ID: float64(1)})
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T", resp.Result)
	}
	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatalf("tools type = %T", result["tools"])
	}
	if len(tools) != 3 {
		t.Errorf("tools count = %d, want 3", len(tools))
	}
	expected := map[string]bool{"describe_image_file": false, "describe_clipboard_image": false, "describe_base64_image": false}
	for _, td := range tools {
		name, _ := td["name"].(string)
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing tool %q in tools/list", name)
		}
	}
}

func TestHandleRequest_MethodNotFound(t *testing.T) {
	s, _ := newTestServer(t)
	s.initMu.Lock()
	s.initialized = true
	s.initMu.Unlock()
	resp := s.handleRequest(nil, JSONRPCRequest{ID: float64(1), Method: "bogus/method"})
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Errorf("expected -32601 method not found, got %+v", resp.Error)
	}
}
