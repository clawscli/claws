# 配置

## AWS 凭证

claws 使用标准的 AWS 配置：

- `~/.aws/credentials` - AWS 凭证
- `~/.aws/config` - AWS 配置（区域、配置文件）
- 环境变量：`AWS_PROFILE`、`AWS_REGION`、`AWS_ACCESS_KEY_ID` 等

## 配置文件

可选设置可以保存在 `~/.config/claws/config.yaml` 中。

### 自定义配置文件路径

使用自定义配置文件代替默认配置：

```bash
# 通过 CLI 标志指定
claws -c /path/to/config.yaml
claws --config ~/work/claws-work.yaml

# 通过环境变量指定
CLAWS_CONFIG=/path/to/config.yaml claws
```

**优先级：** `-c` 标志 > `CLAWS_CONFIG` 环境变量 > 默认路径 (`~/.config/claws/config.yaml`)

使用场景：
- 按环境区分配置（工作/个人）
- 在 CI/CD 中使用项目专属设置
- 使用不同配置进行测试

### 配置文件格式

```yaml
timeouts:
  aws_init: 10s           # AWS 初始化超时（默认：5s）
  multi_region_fetch: 60s # 多区域并行获取超时（默认：30s）
  tag_search: 45s         # 标签搜索超时（默认：30s）
  metrics_load: 30s       # CloudWatch 指标加载超时（默认：30s）
  log_fetch: 15s          # CloudWatch Logs 获取超时（默认：10s）

concurrency:
  max_fetches: 100        # 最大并发 API 获取数（默认：50）

cloudwatch:
  window: 15m             # 指标数据窗口周期（默认：15m）

autosave:
  enabled: true           # 区域/配置文件/主题/compact_header 变更时自动保存（默认：false）

compact_header: false     # 使用单行紧凑标题栏（默认：false）

startup:                  # 启动时应用（如已配置）
  view: services          # 启动视图："dashboard"、"services" 或 "service/resource"（如 "ec2"、"rds/snapshots"）
  filter: bastion         # 启动时应用的模糊筛选（相当于按 `/`）；CLI -f/--filter 优先
  tag: Role=bastion       # 启动时应用的标签筛选（相当于 `:tag`）；CLI --tag 优先
  profiles:               # 支持多个配置文件
    - production
  regions:
    - us-east-1
    - us-west-2

navigation:
  max_stack_size: 100     # 导航历史最大深度（默认：100）

ai:
  profile: ""                  # Bedrock 使用的 AWS 配置文件（留空 = 使用当前配置文件）
  region: ""                   # Bedrock 使用的 AWS 区域（留空 = 使用当前区域）
  model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"  # Bedrock 模型 ID
  max_sessions: 100            # 最大保存会话数（默认：100）
  max_tokens: 16000            # 最大响应 token 数（默认：16000）
  thinking_budget: 8000        # 扩展思考 token 预算（默认：8000）
  max_tool_rounds: 15          # 每条消息的最大工具执行轮数（默认：15）
  max_tool_calls_per_query: 50 # 每次用户查询的最大工具调用数（默认：50）
  save_sessions: false         # 将聊天会话持久化到磁盘（默认：false）

theme: nord               # 预设主题：dark、light、nord、dracula、gruvbox、catppuccin

# 使用预设主题并自定义覆盖：
# theme:
#   preset: dracula
#   primary: "#ff79c6"
#   danger: "#ff5555"
```

配置文件**不会自动创建**，如有需要请手动创建。

CLI 标志（`-p`、`-r`、`-t`、`--compact`、`--no-compact`、`--autosave`、`--no-autosave`）会覆盖配置文件中的设置。
支持多个值：`-p dev,prod` 或 `-p dev -p prod`。

### 特殊配置文件 ID

| ID | 说明 | 等效操作 |
|----|------|----------|
| `__sdk_default__` | 使用 AWS SDK 默认凭证链 | （不使用 `-p` 标志） |
| `__env_only__` | 忽略 ~/.aws，仅使用环境变量/IMDS/ECS/Lambda 凭证 | `-e` 标志 |

```bash
# 通过 -p 标志使用仅环境变量模式
claws -p __env_only__

# 将命名配置文件与特殊模式组合使用（同时查询两者）
claws -p production,__env_only__
```

这些 ID 也可以在 `startup.profiles` 中使用：

```yaml
startup:
  profiles:
    - __sdk_default__
    - production
```


## 主题

claws 内置了 6 种配色主题：

| 主题 | 说明 |
|------|------|
| `dark` | 默认深色主题（粉色/品红色调） |
| `light` | 适用于浅色背景终端 |
| `nord` | 北欧风格，沉稳蓝色调 |
| `dracula` | 流行的深色主题（紫色/粉色） |
| `gruvbox` | 复古暖色调 |
| `catppuccin` | 现代柔和色调（Mocha 变体） |

### 主题预览

| dark | light | nord |
|------|-------|------|
| ![dark](images/theme-dark.png) | ![light](images/theme-light.png) | ![nord](images/theme-nord.png) |

| dracula | gruvbox | catppuccin |
|---------|---------|------------|
| ![dracula](images/theme-dracula.png) | ![gruvbox](images/theme-gruvbox.png) | ![catppuccin](images/theme-catppuccin.png) |

### 切换主题

```bash
# 通过命令行指定
claws -t nord

# 通过命令模式切换（运行时）
:theme dracula
```

如果启用了 autosave，主题更改会自动保存到配置文件中。

### 自定义主题颜色

覆盖预设主题中的特定颜色：

```yaml
theme:
  preset: dracula
  primary: "#ff79c6"
  danger: "#ff5555"
  success: "#50fa7b"
```

## 只读模式

禁用所有破坏性操作：

```bash
# 通过标志指定
claws --read-only

# 通过环境变量指定
CLAWS_READ_ONLY=1 claws
```

## 调试日志

启用调试日志输出到文件：

```bash
claws -l debug.log
```

## IAM 权限

有关所需的 IAM 权限，请参阅 [IAM 权限](iam-permissions.zh-CN.md)。
