# AI Chat

AI Chat provides an intelligent assistant that helps you analyze AWS resources, compare configurations, identify security risks, and navigate documentation.

## Overview

Press `A` in the following views to open AI Chat:
- **Resource Browser** (list view) - Analyzes visible resources
- **Detail View** - Analyzes the selected resource
- **Diff View** - Compares two resources side-by-side

The assistant has access to:
- Current resource context (what you're viewing)
- Active AWS profile and region
- Tools to query resources, fetch logs, and search AWS documentation

## Setup

### 1. IAM Permissions

The AI Chat feature uses Amazon Bedrock. You need the following permission:

```json
{
  "Effect": "Allow",
  "Action": "bedrock:InvokeModelWithResponseStream",
  "Resource": "arn:aws:bedrock:*::foundation-model/*"
}
```

See [IAM Permissions](iam-permissions.md#ai-chat-optional) for details.

### 2. Configuration

Configure AI Chat in `~/.config/claws/config.yaml`:

```yaml
ai:
  profile: ""                  # AWS profile for Bedrock (empty = use current profile)
  region: ""                   # AWS region for Bedrock (empty = use current region)
  model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"  # Bedrock model ID
  max_sessions: 100            # Max stored sessions (default: 100)
  max_tokens: 16000            # Max response tokens (default: 16000)
  thinking_budget: 8000        # Extended thinking token budget (default: 8000)
  max_tool_rounds: 15          # Max tool execution rounds per message (default: 15)
  max_tool_calls_per_query: 50 # Max tool calls per user query (default: 50)
  save_sessions: false         # Persist chat sessions to disk (default: false)
```

See [Configuration](configuration.md) for all options.

## Usage

### Opening Chat

Press `A` in list/detail/diff views to open the AI Chat overlay.

### What the AI Can Do

- List and query AWS resources across services and regions
- Get detailed information about specific resources
- Fetch CloudWatch logs for supported resources (Lambda, ECS, CodeBuild, etc.)
- Search AWS documentation

The AI automatically uses the current profile, region, and resource context from your view.

### Context Awareness

The assistant automatically receives context based on your current view:

**Resource Browser (List View)**:
```
Currently viewing: ec2/instances (us-west-2, production profile)
Visible resources: [i-abc123, i-def456, ...]
```

**Detail View**:
```
Currently viewing: ec2/instances/i-abc123 (us-west-2, production profile)
Resource details: {...}
```

**Diff View**:
```
Comparing two resources:
Left: ec2/instances/i-abc123
Right: ec2/instances/i-def456
```

### Session History

Press `Ctrl+H` to view and resume previous chat sessions.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `A` | Open AI Chat (in list/detail/diff views) |
| `Ctrl+H` | Session history |
| `Enter` | Send message |
| `Esc` | Close chat / Cancel stream |
| `Ctrl+C` | Cancel stream |

## Extended Thinking

The assistant supports extended thinking for complex queries. When enabled, you'll see a thinking indicator showing the assistant's reasoning process before the final response.

Configure thinking budget in config.yaml:
```yaml
ai:
  thinking_budget: 8000  # Max tokens for extended thinking (default: 8000)
```

## Troubleshooting

### "Bedrock not available in this region"

Bedrock is not available in all AWS regions. Configure a supported region in your config:

```yaml
ai:
  region: "us-west-2"  # Use a region where Bedrock is available
```

### "Access Denied" errors

Ensure your IAM role/user has the required Bedrock permissions. See [IAM Permissions](iam-permissions.md#ai-chat-optional).

### Tool call limit reached

If you see "Tool call limit reached", the assistant made too many tool calls in a single query. Increase the limit:

```yaml
ai:
  max_tool_calls_per_query: 100  # Increase from default 50
```

### Session not persisting

Enable session persistence in config:

```yaml
ai:
  save_sessions: true  # Default: false
```

Sessions are stored in `~/.config/claws/sessions/`.
