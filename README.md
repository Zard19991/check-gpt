# API 中转链路检测工具

[![Tests](https://github.com/go-coders/check-trace/actions/workflows/test.yml/badge.svg)](https://github.com/go-coders/check-trace/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-coders/check-trace)](https://goreportcard.com/report/github.com/go-coders/check-trace)

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

请输入API信息:

API URL: https://example-api.com/v1/chat/completions
API Key: sk-xxxx
模型名称 (默认: gpt-4o):

=== API 中转链路检测工具 ===
临时域名: https://hello-world-example.lhr.life
API URL: https://example-api.com/v1/chat/completions
API Key: sk-xxxx***
模型名称: gpt-4o

正在检测中...
   节点1  : Go代理             IP: 1.1.1.1 (Chicago - Cloudflare)
   节点2  : Go代理             IP: 2.2.2.2 (San Francisco - Cloudflare)
   节点3  : OpenAI            IP: 3.3.3.3 (Los Angeles - Microsoft)

检测完成

```
