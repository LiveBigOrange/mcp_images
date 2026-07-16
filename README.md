# mcp_images

**MCP 图片分析服务器** — 在 AI 工具中直接分析本地图片、剪贴板截图或 Base64 图片数据。

将图片发送至兼容 OpenAI 的 VLM API（Ollama / vLLM / DashScope / OpenAI 等），返回结构化分析结果。

## 快速开始

### 方式一：下载预编译二进制（推荐）

从 [Releases](https://github.com/LiveBigOrange/mcp_images/releases/latest) 下载对应平台的二进制文件。

### 方式二：从源码编译

```bash
# Linux / macOS
make build                # 编译当前平台
make build-all            # 跨平台编译（Windows/Linux/macOS）

# Windows
./build.ps1               # 编译当前平台
./build.ps1 -Action build-all  # 跨平台编译
```

编译产物输出到 `bin/` 目录。

### 配置

在 AI 工具的 `mcp.json` 中添加（以 Ollama 为例）：

```json
{
  "mcpServers": {
    "mcp_images": {
      "command": "/path/to/mcp_images",
      "env": {
        "VLM_API_BASE": "http://localhost:11434/v1/chat/completions",
        "VLM_MODEL": "qwen2.5vl:7b"
      }
    }
  }
}
```

更多配置示例（vLLM / DashScope / OpenAI / opencode）见 [完整文档](https://livebigorange.github.io/mcp_images/)。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VLM_API_BASE` | VLM API 地址（必填） | — |
| `VLM_MODEL` | 模型名（必填） | — |
| `VLM_API_KEY` | API Key（本地模型可空） | — |
| `VLM_TIMEOUT` | HTTP 超时秒数 | `60` |
| `VLM_LOG_LEVEL` | 日志级别 | `warn` |

## 工具

- `describe_image_file` — 分析本地图片文件
- `describe_clipboard_image` — 读取剪贴板截图并分析
- `describe_base64_image` — 分析 Base64 编码的图片

## 文档

完整文档（含多场景配置示例、使用案例）：[livebigorange.github.io/mcp_images](https://livebigorange.github.io/mcp_images/)

## 构建

```bash
# Linux / macOS
make build       # 编译当前平台
make build-all   # 跨平台编译
make test        # 运行测试
make lint        # 代码检查

# Windows
./build.ps1               # 编译当前平台
./build.ps1 -Action test  # 运行测试
./build.ps1 -Action lint  # 代码检查
```

## 架构

```
AI 工具 → stdio JSON-RPC → 图片处理 → VLM API → 结构化结果
```

Go 单二进制 + stdio JSON-RPC，零依赖运行。

## 联系

如有问题或建议，请联系 [yzn5555@163.com](mailto:yzn5555@163.com)

## 许可证

[MIT](./LICENSE)
