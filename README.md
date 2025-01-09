# Check-GPT

[![Tests](https://github.com/go-coders/check-gpt/actions/workflows/test.yml/badge.svg)](https://github.com/go-coders/check-gpt/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-coders/check-gpt)](https://goreportcard.com/report/github.com/go-coders/check-gpt)

chatgpt, claude 等大模型中转服务/代理的检测工具，欢迎 star，未来会加入更多实用功能

## 功能特性

- [x] api key 可用性，速度测试
- [x] chatgpt 中转/代理节点检测
- [ ] api key 存储/管理
- [ ] 计费管理
- [ ] ...

## 安装方法

### 方式一：安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/go-coders/check-gpt/main/install.sh | bash
```

### 方式二：Go Install

```bash
go install github.com/go-coders/cmd/check-gpt@latest
```

## 功能特性

### 1. API Key 可用性测试

通过向 gpt 发送简单请求，检测 API Key 是否可用

示例输出：

```
🔍 API 测试信息
--------------------------------------------------------------------------------
API URL:  https://api.example.com/v1/chat/completions
API Keys: sk-abcdefaaaa, sk-abcdefbbbb
模型: gpt-3.5-turbo, gpt-4o, gpt-4o-mini, claude-3-5-sonnet-20241022

🌐 测试中,请稍等...

🚀 测试结果
--------------------------------------------------------------------------------
[1] sk-abcdefaaaa
│ 状态: 🎉 全部可用
│ 模型:
│   gpt-3.5-turbo              ✅ 0.72s
│   gpt-4o                     ✅ 3.89s
│   gpt-4o-mini                ✅ 0.94s
│   claude-3-5-sonnet-20241022 ✅ 0.97s

[2] sk-abcdefbbbb
│ 状态: ⭐ 3/4可用
│ 模型:
│   gpt-3.5-turbo              ✅ 0.55s
│   gpt-4o                     ✅ 3.12s
│   gpt-4o-mini                ✅ 0.89s
│   claude-3-5-sonnet-20241022 ❌

⚙️ 错误信息
--------------------------------------------------------------------------------
❌ [2] sk-abcdefbbbb: [claude-3-5-sonnet-20241022] 当前令牌分组下对于模型 claude-3-5-sonnet-20241022 无可用渠道

```

### 2. API 中转链路检测

1. 向 gpt 发送一个带图片的请求，检测多少个代理请求了图片
2. 使用 localhost.run 临时启动一个图片服务器

示例输出：

```
🔍 API 测试信息
--------------------------------------------------------------------------------
API URL:  https://api.example.com/v1/chat/completions
API Keys: sk-abcdefaaaa
模型: gpt-4o
临时图片URL: https://temp.example.com/image?id=abc123

🌐 测试中,请稍等...

🔗 节点链路
--------------------------------------------------------------------------------
   节点 1 : Go服务               IP: 1.2.3.4 (Tokyo,Japan - AWS Cloud)
   节点 2 : Go服务               IP: 1.2.3.5 (Washington,United States - Cloudflare,Inc)
   节点 3 : 可能是OpenAI服务 💎   IP: 1.2.3.6 (California,United States - Microsoft Azure Cloud)

⚙️ 请求响应
--------------------------------------------------------------------------------
请求: what's the number? (发送验证码图片，验证码: 1234)
响应: The number is 1234.
```
