# AI 聊天

AI 聊天提供了一个智能助手，帮助您分析 AWS 资源、比较配置、识别安全风险以及搜索文档。

## 概述

在以下视图中按 `A` 可打开 AI 聊天：
- **资源浏览器**（列表视图） - 分析当前可见的资源
- **详细视图** - 分析选中的资源
- **差异视图** - 并排比较两个资源

助手可以访问以下信息：
- 当前资源上下文（您正在查看的内容）
- 当前使用的 AWS 配置文件和区域
- 查询资源、获取日志和搜索 AWS 文档的工具

## 设置

### 1. IAM 权限

AI 聊天功能使用 Amazon Bedrock。您需要以下权限：

```json
{
  "Effect": "Allow",
  "Action": "bedrock:InvokeModelWithResponseStream",
  "Resource": "arn:aws:bedrock:*::foundation-model/*"
}
```

详情请参阅 [IAM 权限](iam-permissions.zh-CN.md#ai-聊天可选)。

### 2. 配置

在 `~/.config/claws/config.yaml` 中配置 AI 聊天：

```yaml
ai:
  profile: ""                  # Bedrock 的 AWS 配置文件（空 = 使用当前配置文件）
  region: ""                   # Bedrock 的 AWS 区域（空 = 使用当前区域）
  model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"  # Bedrock 模型 ID
  max_sessions: 100            # 最大保存会话数（默认：100）
  max_tokens: 16000            # 最大响应令牌数（默认：16000）
  thinking_budget: 8000        # 扩展思考令牌预算（默认：8000）
  max_tool_rounds: 15          # 每条消息的最大工具执行轮数（默认：15）
  max_tool_calls_per_query: 50 # 每次查询的最大工具调用次数（默认：50）
  save_sessions: false         # 将聊天会话持久化到磁盘（默认：false）
```

所有选项请参阅 [配置](configuration.zh-CN.md)。

## 使用

### 打开聊天

在列表/详细/差异视图中按 `A` 可打开 AI 聊天覆盖层。

### AI 可以做什么

- 跨服务和区域列出和查询 AWS 资源
- 获取特定资源的详细信息
- 获取支持的资源（Lambda、ECS、CodeBuild 等）的 CloudWatch 日志
- 使用通用且可公开的 AWS 术语搜索 AWS 文档

AI 会自动使用当前视图中的配置文件、区域和资源上下文。

AWS 文档搜索会将搜索查询发送到 AWS 的公共文档搜索端点。为避免泄露私有 AWS 上下文，claws 会拒绝包含资源 ID、账户 ID、ARN、配置文件名称、日志行、标签值或密钥的文档搜索查询。

### 上下文感知

助手会根据您当前的视图自动接收上下文信息：

**资源浏览器（列表视图）**：
```
Currently viewing: ec2/instances (us-west-2, production profile)
Visible resources: [i-abc123, i-def456, ...]
```

**详细视图**：
```
Currently viewing: ec2/instances/i-abc123 (us-west-2, production profile)
Resource details: {...}
```

**差异视图**：
```
Comparing two resources:
Left: ec2/instances/i-abc123
Right: ec2/instances/i-def456
```

### 会话历史

按 `Ctrl+H` 可查看和恢复之前的聊天会话。

## 键盘快捷键

| 按键 | 操作 |
|------|------|
| `A` | 打开 AI 聊天（在列表/详细/差异视图中） |
| `Ctrl+H` | 会话历史 |
| `Enter` | 发送消息 |
| `Esc` | 关闭聊天 / 取消流式输出 |
| `Ctrl+C` | 取消流式输出 |

## 扩展思考

助手支持对复杂查询进行扩展思考。启用后，在最终回答之前会显示一个思考指示器，展示助手的推理过程。

在 config.yaml 中配置思考预算：
```yaml
ai:
  thinking_budget: 8000  # 扩展思考的最大令牌数（默认：8000）
```

## 故障排除

### "Bedrock not available in this region"

Bedrock 并非在所有 AWS 区域都可用。请在配置中指定一个支持的区域：

```yaml
ai:
  region: "us-west-2"  # 使用 Bedrock 可用的区域
```

### "Access Denied" 错误

请确保您的 IAM 角色/用户具有所需的 Bedrock 权限。请参阅 [IAM 权限](iam-permissions.zh-CN.md#ai-聊天可选)。

### 工具调用次数达到上限

如果看到 "Tool call limit reached"，说明助手在单次查询中进行了过多的工具调用。请增加上限：

```yaml
ai:
  max_tool_calls_per_query: 100  # 从默认的 50 增加
```

### 会话未持久化

在配置中启用会话持久化：

```yaml
ai:
  save_sessions: true  # 默认：false
```

会话存储在 `~/.config/claws/chat/sessions/` 中。
