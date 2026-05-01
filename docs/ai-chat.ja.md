# AIチャット

AIチャットは、AWSリソースの分析、設定の比較、セキュリティリスクの特定、ドキュメントの検索を支援するインテリジェントなアシスタントです。

## 概要

以下のビューで`A`を押すとAIチャットが開きます：
- **リソースブラウザ**（リストビュー） - 表示中のリソースを分析します
- **詳細ビュー** - 選択したリソースを分析します
- **差分ビュー** - 2つのリソースを並べて比較します

アシスタントは以下の情報にアクセスできます：
- 現在のリソースコンテキスト（表示中の内容）
- アクティブなAWSプロファイルとリージョン
- リソースのクエリ、ログの取得、AWSドキュメントの検索を行うツール

## セットアップ

### 1. IAM権限

AIチャット機能はAmazon Bedrockを使用します。以下の権限が必要です：

```json
{
  "Effect": "Allow",
  "Action": "bedrock:InvokeModelWithResponseStream",
  "Resource": "arn:aws:bedrock:*::foundation-model/*"
}
```

詳細は[IAM権限](iam-permissions.ja.md#aiチャットオプション)を参照してください。

### 2. 設定

`~/.config/claws/config.yaml`でAIチャットを設定します：

```yaml
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
```

すべてのオプションについては[設定](configuration.ja.md)を参照してください。

## 使い方

### チャットを開く

リスト/詳細/差分ビューで`A`を押すとAIチャットオーバーレイが開きます。

### AIができること

- サービスやリージョンをまたいでAWSリソースの一覧取得やクエリを実行
- 特定のリソースの詳細情報を取得
- 対応リソース（Lambda、ECS、CodeBuildなど）のCloudWatchログを取得
- AWSドキュメントを検索

AIは現在のプロファイル、リージョン、リソースコンテキストを自動的に使用します。

### コンテキスト認識

アシスタントは現在のビューに基づいてコンテキストを自動的に受け取ります：

**リソースブラウザ（リストビュー）**：
```
Currently viewing: ec2/instances (us-west-2, production profile)
Visible resources: [i-abc123, i-def456, ...]
```

**詳細ビュー**：
```
Currently viewing: ec2/instances/i-abc123 (us-west-2, production profile)
Resource details: {...}
```

**差分ビュー**：
```
Comparing two resources:
Left: ec2/instances/i-abc123
Right: ec2/instances/i-def456
```

### セッション履歴

`Ctrl+H`を押すと、過去のチャットセッションを表示・再開できます。

## キーボードショートカット

| キー | アクション |
|------|-----------|
| `A` | AIチャットを開く（リスト/詳細/差分ビュー） |
| `Ctrl+H` | セッション履歴 |
| `Enter` | メッセージを送信 |
| `Esc` | チャットを閉じる / ストリームをキャンセル |
| `Ctrl+C` | ストリームをキャンセル |

## 拡張思考

アシスタントは複雑なクエリに対して拡張思考をサポートしています。有効にすると、最終的な回答の前にアシスタントの推論プロセスを示す思考インジケーターが表示されます。

config.yamlで思考予算を設定します：
```yaml
ai:
  thinking_budget: 8000  # 拡張思考の最大トークン数（デフォルト: 8000）
```

## トラブルシューティング

### 「Bedrock not available in this region」

BedrockはすべてのAWSリージョンで利用できるわけではありません。設定でサポートされているリージョンを指定してください：

```yaml
ai:
  region: "us-west-2"  # Bedrockが利用可能なリージョンを使用
```

### 「Access Denied」エラー

IAMロール/ユーザーに必要なBedrock権限があることを確認してください。[IAM権限](iam-permissions.ja.md#aiチャットオプション)を参照してください。

### ツール呼び出し制限に達した場合

「Tool call limit reached」と表示された場合、アシスタントが1回のクエリで多くのツール呼び出しを行いました。制限を引き上げてください：

```yaml
ai:
  max_tool_calls_per_query: 100  # デフォルトの50から引き上げ
```

### セッションが保持されない場合

設定でセッションの永続化を有効にしてください：

```yaml
ai:
  save_sessions: true  # デフォルト: false
```

セッションは`~/.config/claws/chat/sessions/`に保存されます。
