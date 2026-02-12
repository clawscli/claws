[🇬🇧 English](README.md) | [🇨🇳 简体中文](README.zh-CN.md) | [🇰🇷 한국어](README.ko.md)

# claws

AWSリソース管理のためのターミナルUI

[![CI](https://github.com/clawscli/claws/actions/workflows/ci.yml/badge.svg)](https://github.com/clawscli/claws/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/clawscli/claws)](https://github.com/clawscli/claws/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/clawscli/claws)](https://goreportcard.com/report/github.com/clawscli/claws)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

![claws デモ](docs/images/demo.gif)

## 機能

- **インタラクティブTUI** - vimスタイルのキーバインドでAWSリソースを操作できます
- **70サービス、175リソース** - EC2、S3、Lambda、RDS、ECS、EKSなど多数に対応しています
- **マルチプロファイル＆マルチリージョン** - 複数のアカウント/リージョンを並列でクエリできます
- **リソースアクション** - インスタンスの起動/停止、リソースの削除、ログのテールが可能です
- **クロスリソースナビゲーション** - VPCからサブネット、LambdaからCloudWatchへジャンプできます
- **フィルタリング＆ソート** - あいまい検索、タグフィルタリング、カラムソートに対応しています
- **リソース比較** - サイドバイサイドの差分ビューで比較できます
- **AIチャット** - AWSコンテキスト対応のAIアシスタント（Bedrock経由）
- **6種類のカラーテーマ** - dark、light、nord、dracula、gruvbox、catppuccin

## スクリーンショット

| リソースブラウザ | 詳細ビュー | アクションメニュー |
|------------------|------------|-------------------|
| ![browser](docs/images/resource-browser.png) | ![detail](docs/images/detail-view.png) | ![actions](docs/images/actions-menu.png) |

### マルチリージョン＆マルチアカウント

![multi-region](docs/images/multi-account-region.png)

### AIチャット（Bedrock）

![ai-chat](docs/images/ai-chat.png)

リスト/詳細/差分ビューで`A`を押すとAIチャットが開きます。アシスタントがAWS Bedrockを使用してリソースの分析、設定の比較、リスクの特定を行います。

## インストール

### Homebrew（macOS/Linux）

```bash
brew install --cask clawscli/tap/claws
```

### インストールスクリプト（macOS/Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/clawscli/claws/main/install.sh | sh
```

### バイナリのダウンロード

[GitHub Releases](https://github.com/clawscli/claws/releases/latest)からダウンロードできます。

### Go Install

```bash
go install github.com/clawscli/claws/cmd/claws@latest
```

## クイックスタート

```bash
# claws を実行（デフォルトのAWS認証情報を使用）
claws

# プロファイルを指定
claws -p myprofile

# リージョンを指定
claws -r us-west-2

# サービスやビューを指定して起動
claws -s dashboard        # ダッシュボードから開始
claws -s services         # サービスブラウザから開始（デフォルト）
claws -s ec2              # EC2インスタンス
claws -s rds/snapshots    # RDSスナップショット

# 複数のプロファイル/リージョン（カンマ区切りまたは繰り返し指定）
claws -p dev,prod -r us-east-1,ap-northeast-1

# 読み取り専用モード（破壊的なアクションを無効化）
claws --read-only
```

## キーバインド

| キー | アクション |
|------|-----------|
| `j` / `k` | 上下に移動します |
| `Enter` / `d` | リソースの詳細を表示します |
| `:` | コマンドモード（例: `:ec2/instances`） |
| `/` | フィルターモード（あいまい検索） |
| `a` | アクションメニューを開きます |
| `A` | AIチャット（リスト/詳細/差分ビュー） |
| `R` | リージョンを選択します |
| `P` | プロファイルを選択します |
| `?` | ヘルプを表示します |
| `q` | 終了します |

詳細は[キーバインド](docs/keybindings.ja.md)を参照してください。

## ドキュメント

| ドキュメント | 説明 |
|-------------|------|
| [キーバインド](docs/keybindings.ja.md) | キーボードショートカットの完全なリファレンス |
| [対応サービス](docs/services.ja.md) | 全70サービスと175リソース |
| [設定](docs/configuration.ja.md) | 設定ファイル、テーマ、オプション |
| [IAM権限](docs/iam-permissions.ja.md) | 必要なAWS権限 |
| [AIチャット](docs/ai-chat.ja.md) | AIアシスタントの使い方と機能 |
| [Architecture](docs/architecture.md) | 内部設計と構造 |
| [Adding Resources](docs/adding-resources.md) | コントリビューター向けガイド |

## 開発

### 前提条件

- Go 1.25+
- [Task](https://taskfile.dev/)（オプション）

### コマンド

```bash
task build          # バイナリをビルド
task run            # アプリケーションを実行
task test           # テストを実行
task lint           # リンターを実行
```

## 技術スタック

- **TUI**: [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **AWS**: [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

## ライセンス

Apache License 2.0 - 詳細は[LICENSE](LICENSE)を参照してください。
