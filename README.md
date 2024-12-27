# API 中转链路检测工具

一个用于检测 API 中转链路的命令行工具。通过分析请求链路上的代理节点，帮助您了解 API 请求的转发路径。

## 安装

使用 `curl` 安装：

```bash
curl -fsSL https://raw.githubusercontent.com/go-coders/check-trace/main/install.sh | bash
```

使用 go 安装：

```bash
go install github.com/go-coders/check-trace@latest
```

## 使用示例

```bash
check-trace

正在启动服务器和创建临时域名...
请稍候...

临时域名: https://hello-world-example.lhr.life

=== API 中转链路检测工具 ===

请输入API信息:

API URL: https://example-api.com/v1/chat/completions
API Key: sk-xxxx1234yyyy5678zzzz9012
模型名称 (默认: gpt-4o):

=== API 中转链路检测工具 ===
临时域名: https://hello-world-example.lhr.life
API URL: https://example-api.com/v1/chat/completions
API Key: sk-xxxx***
模型名称: gpt-4o

正在检测中...
  → 12:34:56 UA: Go-http-client/1.1   [代理] X-Forwarded-For: 1.2.3.4
  → 12:34:58 UA: Go-http-client/1.1   [代理] X-Forwarded-For: 5.6.7.8
  → 12:35:00 UA: Go-http-client/1.1   [代理] X-Forwarded-For: 9.10.11.12
  → 12:35:01 UA: Java/17.0.8          [代理] X-Forwarded-For: 13.14.15.16
  → 检测完成
```
