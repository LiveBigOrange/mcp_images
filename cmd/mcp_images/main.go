package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"mcp_images/internal/config"
	"mcp_images/internal/logger"
	"mcp_images/internal/server"
)

var version = "dev"
var buildTime = ""

func main() {
	showVersion := flag.Bool("version", false, "显示版本号")
	flag.Parse()

	if *showVersion {
		if buildTime != "" {
			fmt.Printf("mcp_images %s (built at %s)\n", version, buildTime)
		} else {
			fmt.Printf("mcp_images %s\n", version)
		}
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[启动错误] 配置加载失败：%v\n", err)
		os.Exit(1)
	}

	cfg.NormalizeAPIBase()
	cfg.ValidateLogLevel()
	cfg.ValidateTimeout()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	lg := logger.New(cfg.LogLevel)

	if !cfg.IsLocalAPI() && strings.HasPrefix(cfg.APIBase, "http://") {
		lg.Warn("API Base 使用非加密 HTTP 协议，API Key 可能泄露", logger.Field{Key: "apiBase", Value: cfg.APIBase})
	}

	server.Version = version

	srv := server.NewServer(cfg, lg)
	lg.Info("mcp_images 启动", logger.Field{Key: "version", Value: version})

	if err := srv.Run(context.Background()); err != nil {
		lg.Error("服务运行失败", logger.Field{Key: "error", Value: err.Error()})
		os.Exit(1)
	}
}
