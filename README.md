# HTTP内容替换代理服务器

一个用Go语言编写的HTTP代理服务器，能够接收所有类型的HTTP请求，根据配置文件对请求体内容进行关键词替换，然后转发到目标服务器。

## 功能特性

- ✅ 支持所有HTTP方法（GET、POST、PUT、DELETE等）
- ✅ 完整保留原始请求的headers和body结构
- ✅ 支持四种匹配模式：
  - `prefix` - 前缀匹配
  - `suffix` - 后缀匹配  
  - `contains` - 包含匹配
  - `regex` - 正则表达式匹配
- ✅ 支持两种操作：
  - `replace` - 替换内容
  - `delete` - 删除内容
- ✅ YAML格式配置文件，支持注释
- ✅ 配置热重载，无需重启服务器
- ✅ 详细的调试日志，显示原始内容和修改后内容
- ✅ 显示规则匹配情况和处理结果

## 项目结构

```
content_replace/
├── main.go                 # 程序入口
├── go.mod                  # Go模块文件
├── config/
│   ├── config.go           # 配置结构定义和读取
│   └── rules.go            # 替换规则定义
├── proxy/
│   ├── server.go           # HTTP代理服务器
│   ├── handler.go          # 请求处理器
│   └── forwarder.go        # 请求转发器
├── replacer/
│   └── engine.go           # 内容替换引擎
├── logger/
│   └── logger.go           # 日志记录器
├── configs/
│   ├── config.yaml         # 主配置文件
│   └── rules.yaml          # 替换规则配置
└── logs/
    └── proxy.log           # 日志文件
```

## 快速开始

### 1. 编译项目

```bash
go mod tidy
go build -o proxy main.go
```

### 2. 配置服务器

编辑 `configs/config.yaml` 文件：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

target:
  base_url: "http://example.com"  # 目标服务器地址
  timeout: 30s

debug:
  enabled: true              # 启用调试模式
  show_original: true        # 显示原始请求内容
  show_modified: true        # 显示修改后请求内容
  show_rule_matches: true    # 显示规则匹配情况
```

### 3. 配置替换规则

编辑 `configs/rules.yaml` 文件：

```yaml
rules:
  - name: "替换Claude名称"
    enabled: true
    mode: "contains"
    pattern: "You are Claude Code"
    action: "replace"
    value: "AI 助手"

  - name: "删除提醒信息"
    enabled: true
    mode: "contains"
    pattern: "This is a reminder"
    action: "delete"
```

### 4. 启动服务器

```bash
# 使用默认配置文件启动
./proxy

# 指定配置文件路径
./proxy -config /path/to/config.yaml
```

调试模式通过配置文件控制，编辑`configs/config.yaml`中的`debug.enabled`设置。

## 配置说明

### 主配置文件 (config.yaml)

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| server.host | 服务器监听地址 | "0.0.0.0" |
| server.port | 服务器监听端口 | 8080 |
| target.base_url | 目标服务器地址 | - |
| target.timeout | 请求超时时间 | 30s |
| logging.level | 日志级别 | "info" |
| logging.file | 日志文件路径 | "logs/proxy.log" |
| rules.file | 规则文件路径 | "configs/rules.yaml" |
| rules.auto_reload | 是否自动重载规则 | true |
| debug.enabled | 是否启用调试模式 | false |
| debug.show_original | 是否显示原始内容 | true |
| debug.show_modified | 是否显示修改后内容 | true |
| debug.show_rule_matches | 是否显示规则匹配 | true |

### 替换规则配置 (rules.yaml)

每个规则包含以下字段：

| 字段 | 说明 | 必填 |
|------|------|------|
| name | 规则名称 | ✅ |
| enabled | 是否启用规则 | ✅ |
| mode | 匹配模式 (prefix/suffix/contains/regex) | ✅ |
| pattern | 匹配模式 | ✅ |
| action | 动作类型 (replace/delete) | ✅ |
| value | 替换值 (仅replace动作需要) | ❌ |

## 匹配模式说明

### 1. 前缀匹配 (prefix)
```yaml
- name: "替换前缀"
  mode: "prefix"
  pattern: "claude-sonnet"
  action: "replace"
  value: "ai-assistant"
```

### 2. 后缀匹配 (suffix)
```yaml
- name: "替换后缀"
  mode: "suffix"
  pattern: "20250929"
  action: "replace"
  value: "[版本号]"
```

### 3. 包含匹配 (contains)
```yaml
- name: "替换Claude名称"
  mode: "contains"
  pattern: "You are Claude Code"
  action: "replace"
  value: "AI 助手"
```

### 4. 正则匹配 (regex)
```yaml
- name: "匿名化用户ID"
  mode: "regex"
  pattern: "user_[a-f0-9]{64}"
  action: "replace"
  value: "user_anonymous"
```

## 调试模式

启用调试模式后，服务器会输出详细的调试信息：

- 原始请求内容（headers和body）
- 修改后的请求内容
- 规则匹配情况
- 处理结果

调试输出示例：
```
[DEBUG] === 原始请求 ===
[DEBUG] 方法: POST
[DEBUG] 路径: /api/chat
[DEBUG] Body内容:
[DEBUG] {"text": "You are Claude Code"}
[DEBUG] ==================

[DEBUG] 规则匹配: 替换Claude名称 | 模式: contains | 匹配内容: You are Claude Code | 动作: replace | 替换值: AI 助手 | 状态: ✓ 匹配

[DEBUG] === 修改后请求 ===
[DEBUG] Body内容:
[DEBUG] {"text": "AI 助手"}
[DEBUG] ===================
```

## 使用示例

基于您提供的示例文件，以下是实际的替换效果：

### 示例1：替换内容
**原始内容：**
```json
{
  "system": [
    {
      "type": "text",
      "text": "You are Claude Code, Anthropic's official CLI for Claude."
    }
  ]
}
```

**配置规则：**
```yaml
- name: "替换Claude名称"
  mode: "contains"
  pattern: "You are Claude Code"
  action: "replace"
  value: "AI 助手"
```

**修改后内容：**
```json
{
  "system": [
    {
      "type": "text",
      "text": "AI 助手, Anthropic's official CLI for Claude."
    }
  ]
}
```

### 示例2：删除内容
**配置规则：**
```yaml
- name: "删除提醒信息"
  mode: "contains"
  pattern: "This is a reminder that your todo list is currently empty"
  action: "delete"
```

**结果：** 匹配的内容将被完全删除。

## 命令行参数

```bash
./proxy [选项]

选项:
  -config string
        配置文件路径 (默认 "configs/config.yaml")
```

注意：调试模式通过配置文件中的`debug.enabled`字段控制，而不是命令行参数。

## 日志

服务器会在 `logs/proxy.log` 文件中记录所有请求和处理信息。

日志级别：
- `debug`: 详细的调试信息（需要启用debug模式）
- `info`: 一般信息
- `error`: 错误信息

## 性能特性

- 使用Go的并发特性处理大量请求
- 支持HTTP连接池复用
- 内存高效的字符串处理
- 支持配置热重载，无需重启服务

## 注意事项

1. 确保目标服务器地址正确配置
2. 正则表达式需要符合Go的语法规则
3. 大文件传输时注意内存使用
4. 建议在生产环境中关闭调试模式以提高性能

## 许可证

MIT License

## 简单规则模式

为了方便快速配置大量删除和替换规则，项目提供了简单规则模式：

### 基本语法
```yaml
# 删除规则
delete:
  contains:    # 包含匹配删除
    - "要删除的内容1"
    - "要删除的内容2"
  prefix:      # 前缀匹配删除
    - "前缀内容"
  suffix:      # 后缀匹配删除
    - "后缀内容"
  regex:       # 正则匹配删除
    - "正则表达式"

# 替换规则
replace:
  contains:    # 包含匹配替换
    "旧内容1": "新内容1"
    "旧内容2": "新内容2"
  prefix:      # 前缀匹配替换
    "旧前缀": "新前缀"
  suffix:      # 后缀匹配替换
    "旧后缀": "新后缀"
  regex:       # 正则匹配替换
    "正则表达式": "替换内容"
```

### 使用方式

#### 方式1: 在主规则文件中直接定义
```yaml
rules:
  easy:
    delete:
      contains:
        - "广告内容"
        - "垃圾信息"
    replace:
      contains:
        "旧版本": "新版本"
        "bug": "功能"
```

#### 方式2: 引用外部简单规则文件
```yaml
rules:
  easy:
    delete:
      contains:
        - "configs/cleanup_rules.yaml"  # 引用外部文件
```

### 简单规则文件示例
创建 `configs/easy01.yaml`:
```yaml
delete:
  contains:
    - "You are Claude Code"
    - "This is a reminder"
    - "<system-reminder>"
    - "</system-reminder>"

replace:
  contains:
    "This is a": "这是"
    "DO NOT": "不要"
    "Claude Code": "AI助手"
    "Anthropic's official CLI for Claude": "AI命令行工具"
```

### 混合规则配置示例
```yaml
rules:
  # 传统规则
  - name: "自定义复杂规则"
    enabled: true
    mode: "regex"
    pattern: "\\buser_[a-f0-9]{64}\\b"
    action: "replace"
    value: "user_anonymous"
  
  # 简单规则
  easy:
    delete:
      contains:
        - "configs/easy01.yaml"  # 引用外部简单规则文件
```

### 优势
- **快速配置**: 无需为每个规则设置name、enabled等字段
- **批量管理**: 一次性配置多个相似的删除或替换规则
- **外部文件**: 支持将规则分离到独立文件中，便于管理
- **混合使用**: 可以与传统规则模式混合使用
- **自动命名**: 系统自动为简单规则生成描述性名称

### 规则命名规则
简单规则会自动生成名称：
- 删除规则: `批量删除-contains-1`, `批量删除-prefix-1`, 等
- 替换规则: `批量替换-contents-匹配内容前缀`, 等
- 前缀规则: `前缀删除-1`, `前缀替换-匹配内容前缀`, 等
- 后缀规则: `后缀删除-1`, `后缀替换-匹配内容前缀`, 等
- 正则规则: `正则删除-1`, `正则替换-匹配内容前缀`, 等