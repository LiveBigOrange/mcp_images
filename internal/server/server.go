package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"mcp_images/internal/clipboard"
	"mcp_images/internal/config"
	imgproc "mcp_images/internal/image"
	"mcp_images/internal/logger"
	"mcp_images/internal/tool"
	"mcp_images/internal/vlm"
)

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

const maxConcurrentRequests = 16

type MCPServer struct {
	cfg      *config.Config
	logger   logger.Logger
	registry *tool.Registry
	wg       sync.WaitGroup

	writeMu     sync.Mutex
	stdoutW     *bufio.Writer
	sem         chan struct{}
	initialized bool
	initMu      sync.Mutex
}

func NewServer(cfg *config.Config, lg logger.Logger) *MCPServer {
	processor := imgproc.NewProcessorWithMax(cfg.MaxDimension)
	vlmClient := vlm.NewClient(cfg, lg)
	clipReader := clipboard.NewReader(lg)

	registry := tool.NewRegistry()
	registry.Register(tool.NewDescribeImageFile(vlmClient, processor))
	registry.Register(tool.NewDescribeClipboardImage(vlmClient, processor, clipReader))
	registry.Register(tool.NewDescribeBase64Image(vlmClient, processor))

	return &MCPServer{
		cfg:      cfg,
		logger:   lg,
		registry: registry,
		sem:      make(chan struct{}, maxConcurrentRequests),
		stdoutW:  bufio.NewWriterSize(os.Stdout, 64*1024),
	}
}

func (s *MCPServer) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			s.logger.Info("收到终止信号，开始优雅关闭")
			cancel()
		case <-ctx.Done():
		}
	}()

	decoder := json.NewDecoder(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("等待当前请求完成...")
			done := make(chan struct{})
			go func() {
				s.wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				s.logger.Info("所有请求已完成，退出")
			case <-time.After(5 * time.Second):
				s.logger.Warn("等待超时，强制退出")
			}
			return nil
		default:
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				s.logger.Info("stdin 已关闭，5 秒后退出")
				cancel()
				continue
			}
			s.logger.Debug("JSON 解析失败，重建 decoder", logger.Field{Key: "error", Value: err.Error()})
			s.writeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			})
			decoder = json.NewDecoder(os.Stdin)
			continue
		}

		if err := s.dispatchRaw(ctx, raw); err != nil {
			s.logger.Debug("请求分发失败", logger.Field{Key: "error", Value: err.Error()})
		}
	}
}

func (s *MCPServer) dispatchRaw(ctx context.Context, raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil
	}

	if trimmed[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(raw, &batch); err != nil {
			s.writeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error: invalid batch request"},
			})
			return err
		}
		if len(batch) == 0 {
			s.writeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32600, Message: "Invalid Request: empty batch"},
			})
			return nil
		}
		for _, item := range batch {
			_ = s.dispatchRaw(ctx, item)
		}
		return nil
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &RPCError{Code: -32700, Message: "Parse error"},
		})
		return err
	}

	if s.isNotification(req) {
		s.wg.Add(1)
		go func(r JSONRPCRequest) {
			defer s.wg.Done()
			s.handleRequest(ctx, r)
		}(req)
		return nil
	}

	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32603, Message: "Server is shutting down"},
		})
		return ctx.Err()
	}

	s.wg.Add(1)
	go func(r JSONRPCRequest) {
		defer s.wg.Done()
		defer func() { <-s.sem }()
		resp := s.handleRequest(ctx, r)
		s.writeResponse(resp)
	}(req)

	return nil
}

func (s *MCPServer) isNotification(req JSONRPCRequest) bool {
	return req.ID == nil
}

func (s *MCPServer) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		s.initMu.Lock()
		s.initialized = true
		s.initMu.Unlock()
		return s.handleInitialize(req)
	case "notifications/initialized":
		s.logger.Debug("MCP 握手完成")
		return JSONRPCResponse{}
	case "notifications/cancelled":
		s.logger.Debug("收到取消通知")
		return JSONRPCResponse{}
	case "ping":
		return s.handlePing(req)
	case "logging/setLevel":
		if !s.isInitialized() {
			return notInitializedResp(req.ID)
		}
		return s.handleSetLogLevel(req)
	case "tools/list":
		if !s.isInitialized() {
			return notInitializedResp(req.ID)
		}
		return s.handleToolsList(req)
	case "tools/call":
		if !s.isInitialized() {
			return notInitializedResp(req.ID)
		}
		return s.handleToolsCall(ctx, req)
	default:
		if s.isNotification(req) {
			return JSONRPCResponse{}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

func (s *MCPServer) isInitialized() bool {
	s.initMu.Lock()
	defer s.initMu.Unlock()
	return s.initialized
}

func notInitializedResp(id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: -32002, Message: "Server not initialized: send an initialize request first"},
	}
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
				"logging": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp_images",
				"version": Version,
			},
		},
	}
}

func (s *MCPServer) handlePing(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}
}

func (s *MCPServer) handleSetLogLevel(req JSONRPCRequest) JSONRPCResponse {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	var params struct {
		Level string `json:"level"`
	}
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[params.Level] {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid log level: %s (valid: debug, info, warn, error)", params.Level)},
		}
	}

	if stderrLg, ok := s.logger.(*logger.StderrLogger); ok {
		stderrLg.SetLevel(params.Level)
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}
}

func (s *MCPServer) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := s.registry.List()
	toolDefs := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		toolDefs = append(toolDefs, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": t.InputSchema(),
		})
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": toolDefs,
		},
	}
}

func (s *MCPServer) handleToolsCall(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	var callParams ToolCallParams
	if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	t, ok := s.registry.Get(callParams.Name)
	if !ok {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", callParams.Name)},
		}
	}

	args := tool.SanitizeArgs(callParams.Arguments)

	result, err := t.Execute(ctx, args)
	isError := err != nil
	text := result
	if err != nil {
		text = err.Error()
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolResult{
			Content: []ContentBlock{
				{Type: "text", Text: text},
			},
			IsError: isError,
		},
	}
}

func (s *MCPServer) writeResponse(resp JSONRPCResponse) {
	if resp.JSONRPC == "" {
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("响应序列化失败", logger.Field{Key: "error", Value: err.Error()})
		return
	}
	s.writeMu.Lock()
	s.stdoutW.Write(data)
	s.stdoutW.WriteByte('\n')
	s.stdoutW.Flush()
	s.writeMu.Unlock()
}

var Version = "dev"
