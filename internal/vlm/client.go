package vlm

import (
	"context"
	"encoding/json"
	"fmt"
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"mcp_images/internal/config"
	"mcp_images/internal/logger"
)

const systemPrompt = `你是一个资深软件开发助理。请分析这张报错、代码或 UI 截图，并输出极其精准的结构化描述：
1. ERROR_INFO: 提取任何精确的报错文本、调用栈信息、Traceback 或日志。如果不是报错图，请忽略此项。
2. VISUAL_DESC: 用精准的开发术语描述你看到的 UI 问题、样式重叠、网络请求图表、架构设计等视觉表现形式。
3. ENV_CONTEXT: 根据截图特征猜测开发者当前的运行环境（例如：VS Code 编辑器、Linux 终端、Chrome 控制台等）。
请保持输出精炼、直接，不需要任何日常礼貌用语，直接输出这三项分析结果。`

const userTextPrompt = "请分析这张图片。"

const maxRetries = 2
const retryBaseDelay = 1 * time.Second

type VLMContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type VLMMessage struct {
	Role    string       `json:"role"`
	Content []VLMContent `json:"content"`
}

type VLMRequest struct {
	Model     string       `json:"model"`
	Messages  []VLMMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
}

type VLMChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type VLMResponse struct {
	Choices []VLMChoice `json:"choices"`
}

type Client struct {
	cfg    *config.Config
	logger logger.Logger
	http   *http.Client
}

func NewClient(cfg *config.Config, lg logger.Logger) *Client {
	timeout := time.Duration(cfg.Timeout) * time.Second
	transport := &http.Transport{
		MaxIdleConns:      10,
		IdleConnTimeout:   timeout,
		DisableKeepAlives: false,
	}
	return &Client{
		cfg:    cfg,
		logger: lg,
		http: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

func BuildRequest(cfg *config.Config, imageDataURI string) *VLMRequest {
	return &VLMRequest{
		Model: cfg.Model,
		Messages: []VLMMessage{
			{
				Role: "system",
				Content: []VLMContent{
					{Type: "text", Text: systemPrompt},
				},
			},
			{
				Role: "user",
				Content: []VLMContent{
					{Type: "text", Text: userTextPrompt},
					{Type: "image_url", ImageURL: &ImageURL{URL: imageDataURI}},
				},
			},
		},
		MaxTokens: 4096,
	}
}

func (c *Client) AnalyzeImage(ctx context.Context, imageDataURI string) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay << (attempt - 1)
			c.logger.Debug("VLM 请求重试", logger.Field{Key: "attempt", Value: attempt}, logger.Field{Key: "delay", Value: delay.String()})
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		result, err := c.doRequest(ctx, imageDataURI)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return "", err
		}
	}
	return "", lastErr
}

func (c *Client) doRequest(ctx context.Context, imageDataURI string) (string, error) {
	req := BuildRequest(c.cfg, imageDataURI)

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("[VLM错误] 请求构造失败：%v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.cfg.APIBase, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("[VLM错误] 请求创建失败：%v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	c.logger.Debug("发送 VLM 请求", logger.Field{Key: "model", Value: c.cfg.Model}, logger.Field{Key: "url", Value: c.cfg.APIBase})

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("[VLM错误] VLM 服务响应超时。可能是图片过于复杂或服务负载过高，可稍后重试。")
		}
		return "", fmt.Errorf("[VLM错误] 无法连接到 VLM 服务（%v）。请检查 VLM_API_BASE 配置和网络连接。", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("[VLM错误] 读取响应失败：%v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", MapError(nil, resp.StatusCode)
	}

	result, err := ParseResponse(respBody)
	if err != nil {
		return "", err
	}

	return result, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "超时") {
		return true
	}
	if strings.Contains(msg, "429") {
		return true
	}
	if strings.Contains(msg, "500") || strings.Contains(msg, "502") || strings.Contains(msg, "503") {
		return true
	}
	if strings.Contains(msg, "无法连接") {
		return true
	}
	return false
}

func ParseResponse(body []byte) (string, error) {
	var resp VLMResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("[VLM错误] 响应 JSON 解析失败：%v", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("[VLM错误] VLM 返回异常响应：choices 为空。请检查模型是否支持视觉输入。")
	}
	content := resp.Choices[0].Message.Content
	if content == "" {
		return "", fmt.Errorf("[VLM错误] VLM 返回异常响应：content 为空。请检查模型是否支持视觉输入。")
	}
	return content, nil
}

func MapError(err error, statusCode int) error {
	if err != nil {
		return fmt.Errorf("[VLM错误] 无法连接到 VLM 服务（%v）。请检查 VLM_API_BASE 配置和网络连接。", err)
	}
	switch statusCode {
	case 401, 403:
		return fmt.Errorf("[VLM错误] VLM 服务认证失败（HTTP %d）。请检查 VLM_API_KEY 配置是否正确。", statusCode)
	case 404:
		return fmt.Errorf("[VLM错误] VLM 模型不存在（HTTP 404）。请检查 VLM_MODEL 配置是否正确。")
	case 429:
		return fmt.Errorf("[VLM错误] VLM 服务请求频率超限（HTTP 429）。请稍后重试。")
	default:
		if statusCode >= 500 {
			return fmt.Errorf("[VLM错误] VLM 服务内部错误（HTTP %d）。请稍后重试。", statusCode)
		}
		return fmt.Errorf("[VLM错误] VLM 服务返回未知错误（HTTP %d）。", statusCode)
	}
}
