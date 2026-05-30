# 設定

## AWS認証情報

clawsは標準的なAWS設定を使用します：

- `~/.aws/credentials` - AWS認証情報
- `~/.aws/config` - AWS設定（リージョン、プロファイル）
- 環境変数: `AWS_PROFILE`、`AWS_REGION`、`AWS_ACCESS_KEY_ID` など

## 設定ファイル

オプション設定は `~/.config/claws/config.yaml` に保存できます。

### カスタム設定ファイルパス

デフォルトの代わりにカスタム設定ファイルを使用できます：

```bash
# CLIフラグで指定
claws -c /path/to/config.yaml
claws --config ~/work/claws-work.yaml

# 環境変数で指定
CLAWS_CONFIG=/path/to/config.yaml claws
```

**優先順位:** `-c` フラグ > `CLAWS_CONFIG` 環境変数 > デフォルト (`~/.config/claws/config.yaml`)

使用例：
- 環境別の設定（仕事用/個人用）
- プロジェクト固有の設定によるCI/CD
- 異なる設定でのテスト

### 設定ファイルの形式

```yaml
timeouts:
  aws_init: 10s           # AWS初期化タイムアウト（デフォルト: 5s）
  multi_region_fetch: 60s # マルチリージョン並列取得タイムアウト（デフォルト: 30s）
  tag_search: 45s         # タグ検索タイムアウト（デフォルト: 30s）
  metrics_load: 30s       # CloudWatchメトリクス読み込みタイムアウト（デフォルト: 30s）
  log_fetch: 15s          # CloudWatch Logs取得タイムアウト（デフォルト: 10s）

concurrency:
  max_fetches: 100        # 最大同時API取得数（デフォルト: 50）

cloudwatch:
  window: 15m             # メトリクスデータのウィンドウ期間（デフォルト: 15m）

autosave:
  enabled: true           # リージョン/プロファイル/テーマ/compact_headerの変更時に保存（デフォルト: false）

compact_header: false     # 単一行のコンパクトヘッダーを使用（デフォルト: false）

startup:                  # 起動時に適用（設定がある場合）
  view: services          # 起動ビュー: "dashboard"、"services"、または "service/resource"（例: "ec2"、"rds/snapshots"）
  filter: bastion         # 起動時に適用するファジーフィルター（`/` と同等）。CLI の -f/--filter が優先
  tag: Role=bastion       # 起動時に適用するタグフィルター（`:tag` と同等）。CLI の --tag が優先
  profiles:               # 複数プロファイル対応
    - production
  regions:
    - us-east-1
    - us-west-2

navigation:
  max_stack_size: 100     # ナビゲーション履歴の最大深度（デフォルト: 100）

ai:
  profile: ""                  # Bedrock用AWSプロファイル（空 = 現在のプロファイルを使用）
  region: ""                   # Bedrock用AWSリージョン（空 = 現在のリージョンを使用）
  model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"  # BedrockモデルID
  max_sessions: 100            # 最大保存セッション数（デフォルト: 100）
  max_tokens: 16000            # 最大レスポンストークン数（デフォルト: 16000）
  thinking_budget: 8000        # 拡張思考トークン予算（デフォルト: 8000）
  max_tool_rounds: 15          # メッセージあたりの最大ツール実行ラウンド数（デフォルト: 15）
  max_tool_calls_per_query: 50 # ユーザークエリあたりの最大ツール呼び出し数（デフォルト: 50）
  save_sessions: false         # チャットセッションをディスクに永続化（デフォルト: false）

theme: nord               # プリセット: dark, light, nord, dracula, gruvbox, catppuccin

# プリセットにカスタムオーバーライドを適用する場合:
# theme:
#   preset: dracula
#   primary: "#ff79c6"
#   danger: "#ff5555"
```

設定ファイルは**自動的に作成されません**。必要に応じて手動で作成してください。

CLIフラグ（`-p`、`-r`、`-t`、`--compact`、`--no-compact`、`--autosave`、`--no-autosave`）は設定ファイルの値を上書きします。
複数の値を指定できます: `-p dev,prod` または `-p dev -p prod`。

### 特殊プロファイルID

| ID | 説明 | 同等の操作 |
|----|------|-----------|
| `__sdk_default__` | AWS SDKのデフォルト認証チェーンを使用 | （`-p` フラグなし） |
| `__env_only__` | ~/.awsを無視し、環境変数/IMDS/ECS/Lambdaの認証情報のみ使用 | `-e` フラグ |

```bash
# -pフラグで環境変数のみモードを使用
claws -p __env_only__

# 名前付きプロファイルと特殊モードを組み合わせ（両方をクエリ）
claws -p production,__env_only__
```

これらのIDは `startup.profiles` でも使用できます：

```yaml
startup:
  profiles:
    - __sdk_default__
    - production
```


## テーマ

clawsには6つの組み込みカラーテーマがあります：

| テーマ | 説明 |
|--------|------|
| `dark` | デフォルトのダークテーマ（ピンク/マゼンタのアクセント） |
| `light` | 明るい背景のターミナル向け |
| `nord` | 北欧風の落ち着いたブルーパレット |
| `dracula` | 人気のダークテーマ（パープル/ピンク） |
| `gruvbox` | レトロで温かみのあるアーストーン |
| `catppuccin` | モダンなパステルカラー（Mochaバリアント） |

### テーマプレビュー

| dark | light | nord |
|------|-------|------|
| ![dark](images/theme-dark.png) | ![light](images/theme-light.png) | ![nord](images/theme-nord.png) |

| dracula | gruvbox | catppuccin |
|---------|---------|------------|
| ![dracula](images/theme-dracula.png) | ![gruvbox](images/theme-gruvbox.png) | ![catppuccin](images/theme-catppuccin.png) |

### テーマの切り替え

```bash
# コマンドラインで指定
claws -t nord

# コマンドモードで切り替え（実行時）
:theme dracula
```

autosaveが有効な場合、テーマの変更は設定ファイルに保存されます。

### カスタムテーマカラー

プリセットの特定の色をオーバーライドできます：

```yaml
theme:
  preset: dracula
  primary: "#ff79c6"
  danger: "#ff5555"
  success: "#50fa7b"
```

## 読み取り専用モード

すべての破壊的アクションを無効にします：

```bash
# フラグで指定
claws --read-only

# 環境変数で指定
CLAWS_READ_ONLY=1 claws
```

## デバッグログ

ファイルへのデバッグログを有効にします：

```bash
claws -l debug.log
```

## IAM権限

必要なIAM権限については、[IAM権限](iam-permissions.ja.md)を参照してください。
