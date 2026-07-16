package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
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
	Code    int    `json:"code"`
	Message string `json:"message"`
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

type MCPServer struct {
	cfg        *config.Config
	logger     logger.Logger
	registry   *tool.Registry
	wg         sync.WaitGroup
	shutdownMu sync.Mutex
	writeMu    sync.Mutex
	running    bool
}

func NewServer(cfg *config.Config, lg logger.Logger) *MCPServer {
	processor := imgproc.NewProcessor()
	vlmClient := vlm.NewClient(cfg, lg)
	clipReader := clipboard.NewReader(lg)

	registry := tool.NewRegistry()
	registry.Register(tool.NewDescribeImageFile(vlmClient, processor, lg))
	registry.Register(tool.NewDescribeClipboardImage(vlmClient, processor, clipReader, lg))
	registry.Register(tool.NewDescribeBase64Image(vlmClient, processor, lg))

	return &MCPServer{
		cfg:      cfg,
		logger:   lg,
		registry: registry,
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

	s.running = true
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

		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				s.logger.Info("stdin 已关闭，5 秒后退出")
				cancel()
				continue
			}
			s.logger.Debug("JSON 解析失败", logger.Field{Key: "error", Value: err.Error()})
			s.writeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		s.wg.Add(1)
		go func(r JSONRPCRequest) {
			defer s.wg.Done()
			resp := s.handleRequest(ctx, r)
			s.writeResponse(resp)
		}(req)
	}
}

func (s *MCPServer) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		s.logger.Debug("MCP 握手完成")
		return JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
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
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp_images",
				"version": Version,
			},
		},
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
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("响应序列化失败", logger.Field{Key: "error", Value: err.Error()})
		return
	}
	s.writeMu.Lock()
	fmt.Println(string(data))
	s.writeMu.Unlock()
}

var Version = "dev"
