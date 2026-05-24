[🇬🇧 English](README.md) | [🇯🇵 日本語](README.ja.md) | [🇰🇷 한국어](README.ko.md)

# claws

AWS 资源管理终端 UI

[![CI](https://github.com/clawscli/claws/actions/workflows/ci.yml/badge.svg)](https://github.com/clawscli/claws/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/clawscli/claws)](https://github.com/clawscli/claws/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/clawscli/claws)](https://goreportcard.com/report/github.com/clawscli/claws)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

![claws 演示](docs/images/demo.gif)

## 功能

- **交互式 TUI** - 使用 vim 风格的快捷键浏览 AWS 资源
- **70 个服务、175 个资源** - 支持 EC2、S3、Lambda、RDS、ECS、EKS 等众多服务
- **多配置文件与多区域** - 并行查询多个账户和区域
- **配置文件登录辅助** - 可从配置文件选择器刷新 AWS SSO 会话或执行 AWS CLI `aws login`
- **资源操作** - 启动/停止实例、删除资源、追踪日志
- **跨资源导航** - 从 VPC 跳转到子网，从 Lambda 跳转到 CloudWatch
- **筛选与排序** - 模糊搜索、标签筛选、列排序
- **资源比较** - 并排差异对比视图
- **AI 聊天** - 具备 AWS 上下文感知的 AI 助手（通过 Bedrock）
- **6 种配色主题** - dark、light、nord、dracula、gruvbox、catppuccin

## 截图

| 资源浏览器 | 详情视图 | 操作菜单 |
|-----------|---------|---------|
| ![browser](docs/images/resource-browser.png) | ![detail](docs/images/detail-view.png) | ![actions](docs/images/actions-menu.png) |

### 多区域与多账户

![multi-region](docs/images/multi-account-region.png)

### AI 聊天 (Bedrock)

![ai-chat](docs/images/ai-chat.png)

在列表/详情/差异视图中按 `A` 即可打开 AI 聊天。助手会使用 AWS Bedrock 分析资源、比较配置并识别风险。

## 安装

### Homebrew (macOS/Linux)

```bash
brew install --cask clawscli/tap/claws
```

### 安装脚本 (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh | sh
```

### 下载二进制文件

从 [GitHub Releases](https://github.com/clawscli/claws/releases/latest) 下载。

### Go 安装

```bash
go install github.com/clawscli/claws/cmd/claws@latest
```

### Docker

```bash
docker run -it --rm \
  -v ~/.aws:/home/claws/.aws:ro \
  ghcr.io/clawscli/claws
```

## 快速开始

```bash
# 运行 claws（使用默认 AWS 凭证）
claws

# 指定配置文件
claws -p myprofile

# 指定区域
claws -r us-west-2

# 指定服务或视图启动
claws -s dashboard        # 从仪表板开始
claws -s services         # 从服务浏览器开始（默认）
claws -s ec2              # EC2 实例
claws -s rds/snapshots    # RDS 快照

# 多个配置文件/区域（逗号分隔或重复指定）
claws -p dev,prod -r us-east-1,ap-northeast-1

# 只读模式（禁用破坏性操作）
claws --read-only
```

## 键盘快捷键

| 键 | 操作 |
|----|------|
| `j` / `k` | 上下移动 |
| `Enter` / `d` | 查看资源详情 |
| `:` | 命令模式（例如 `:ec2/instances`） |
| `/` | 筛选模式（模糊搜索） |
| `a` | 打开操作菜单 |
| `A` | AI 聊天（列表/详情/差异视图） |
| `R` | 选择区域 |
| `P` | 选择配置文件 |
| `l` | 对选中的配置文件进行 SSO 登录/刷新（配置文件选择器） |
| `L` | 对选中的配置文件执行 AWS CLI `aws login`（配置文件选择器） |
| `?` | 显示帮助 |
| `q` | 退出 |

详细信息请参阅[键盘快捷键](docs/keybindings.zh-CN.md)完整参考。

## 文档

| 文档 | 说明 |
|------|------|
| [键盘快捷键](docs/keybindings.zh-CN.md) | 完整的键盘快捷键参考 |
| [支持的服务](docs/services.zh-CN.md) | 全部 70 个服务和 175 个资源 |
| [配置](docs/configuration.zh-CN.md) | 配置文件、主题和选项 |
| [IAM 权限](docs/iam-permissions.zh-CN.md) | 所需的 AWS 权限 |
| [AI 聊天](docs/ai-chat.zh-CN.md) | AI 助手使用和功能 |
| [Architecture](docs/architecture.md) | 内部设计和架构 |
| [Adding Resources](docs/adding-resources.md) | 贡献者指南 |

## 开发

### 前置要求

- Go 1.25+
- [Task](https://taskfile.dev/)（可选）

### 命令

```bash
task build          # 构建二进制文件
task run            # 运行应用程序
task test           # 运行测试
task lint           # 运行代码检查
```

## 技术栈

- **TUI**: [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **AWS**: [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

## 许可证

Apache License 2.0 - 详情请参阅 [LICENSE](LICENSE)。
