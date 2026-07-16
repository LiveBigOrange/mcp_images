package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIBase  string
	APIKey   string
	Model    string
	LogLevel string
	Timeout  int // VLM HTTP 超时（秒），默认 60
}

func Load() (*Config, error) {
	cfg := &Config{
		APIBase:  strings.TrimSpace(os.Getenv("VLM_API_BASE")),
		APIKey:   strings.TrimSpace(os.Getenv("VLM_API_KEY")),
		Model:    strings.TrimSpace(os.Getenv("VLM_MODEL")),
		LogLevel: strings.TrimSpace(os.Getenv("VLM_LOG_LEVEL")),
		Timeout:  60,
	}
	if t := strings.TrimSpace(os.Getenv("VLM_TIMEOUT")); t != "" {
		if v, err := strconv.Atoi(t); err == nil && v > 0 {
			cfg.Timeout = v
		}
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.APIBase == "" {
		return fmt.Errorf("[适配器未配置] 检测到 VLM_API_BASE 未设置。请在 mcp.json 的 env 中添加：\n\"VLM_API_BASE\": \"https://your-api-endpoint/chat/completions\"\n\"VLM_MODEL\": \"your-vision-model\"")
	}
	if !strings.HasPrefix(c.APIBase, "http://") && !strings.HasPrefix(c.APIBase, "https://") {
		return fmt.Errorf("[适配器配置错误] VLM_API_BASE 必须以 http:// 或 https:// 开头，当前值：%s", c.APIBase)
	}
	if c.Model == "" {
		return fmt.Errorf("[适配器未配置] 检测到 VLM_MODEL 未设置。请在 mcp.json 的 env 中添加：\n\"VLM_MODEL\": \"your-vision-model\"")
	}
	if !c.IsLocalAPI() {
		if c.APIKey == "" {
			return fmt.Errorf("[适配器未配置] VLM_API_BASE 指向云端服务但 VLM_API_KEY 为空。请在 mcp.json 的 env 中添加：\n\"VLM_API_KEY\": \"your-api-key\"")
		}
	}
	return nil
}

func (c *Config) IsLocalAPI() bool {
	return strings.Contains(c.APIBase, "localhost") || strings.Contains(c.APIBase, "127.0.0.1")
}

func (c *Config) NormalizeAPIBase() {
	c.APIBase = strings.TrimRight(c.APIBase, "/")
}

func (c *Config) ValidateTimeout() {
	if c.Timeout <= 0 {
		c.Timeout = 60
	}
}

func (c *Config) ValidateLogLevel() {
	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
		c.LogLevel = strings.ToLower(c.LogLevel)
	case "":
		c.LogLevel = "warn"
	default:
		c.LogLevel = "warn"
	}
}
