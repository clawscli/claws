# AI 채팅

AI 채팅은 AWS 리소스 분석, 설정 비교, 보안 리스크 식별, 문서 검색을 지원하는 인텔리전트 어시스턴트입니다.

## 개요

다음 뷰에서 `A`를 누르면 AI 채팅이 열립니다:
- **리소스 브라우저** (리스트 뷰) - 표시 중인 리소스를 분석합니다
- **상세 뷰** - 선택한 리소스를 분석합니다
- **비교 뷰** - 두 리소스를 나란히 비교합니다

어시스턴트는 다음 정보에 접근할 수 있습니다:
- 현재 리소스 컨텍스트 (표시 중인 내용)
- 활성 AWS 프로필 및 리전
- 리소스 쿼리, 로그 가져오기, AWS 문서 검색을 위한 도구

## 설정

### 1. IAM 권한

AI 채팅 기능은 Amazon Bedrock을 사용합니다. 다음 권한이 필요합니다:

```json
{
  "Effect": "Allow",
  "Action": "bedrock:InvokeModelWithResponseStream",
  "Resource": "arn:aws:bedrock:*::foundation-model/*"
}
```

자세한 내용은 [IAM 권한](iam-permissions.ko.md#ai-채팅-선택-사항)을 참조하십시오.

### 2. 설정

`~/.config/claws/config.yaml`에서 AI 채팅을 설정합니다:

```yaml
ai:
  profile: ""                  # Bedrock용 AWS 프로필 (비어 있으면 현재 프로필 사용)
  region: ""                   # Bedrock용 AWS 리전 (비어 있으면 현재 리전 사용)
  model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"  # Bedrock 모델 ID
  max_sessions: 100            # 최대 저장 세션 수 (기본값: 100)
  max_tokens: 16000            # 최대 응답 토큰 수 (기본값: 16000)
  thinking_budget: 8000        # 확장 사고 토큰 예산 (기본값: 8000)
  max_tool_rounds: 15          # 메시지당 최대 도구 실행 라운드 수 (기본값: 15)
  max_tool_calls_per_query: 50 # 사용자 쿼리당 최대 도구 호출 수 (기본값: 50)
  save_sessions: false         # 채팅 세션을 디스크에 저장 (기본값: false)
```

모든 옵션에 대해서는 [설정](configuration.ko.md)을 참조하십시오.

## 사용법

### 채팅 열기

리스트/상세/비교 뷰에서 `A`를 누르면 AI 채팅 오버레이가 열립니다.

### AI가 할 수 있는 작업

- 서비스 및 리전에 걸쳐 AWS 리소스 목록 조회 및 쿼리 실행
- 특정 리소스의 상세 정보 가져오기
- 지원 리소스(Lambda, ECS, CodeBuild 등)의 CloudWatch 로그 가져오기
- AWS 문서 검색

AI는 현재 프로필, 리전, 리소스 컨텍스트를 자동으로 사용합니다.

### 컨텍스트 인식

어시스턴트는 현재 뷰에 따라 컨텍스트를 자동으로 수신합니다:

**리소스 브라우저 (리스트 뷰)**:
```
Currently viewing: ec2/instances (us-west-2, production profile)
Visible resources: [i-abc123, i-def456, ...]
```

**상세 뷰**:
```
Currently viewing: ec2/instances/i-abc123 (us-west-2, production profile)
Resource details: {...}
```

**비교 뷰**:
```
Comparing two resources:
Left: ec2/instances/i-abc123
Right: ec2/instances/i-def456
```

### 세션 기록

`Ctrl+H`를 누르면 이전 채팅 세션을 확인하고 재개할 수 있습니다.

## 키보드 단축키

| 키 | 액션 |
|----|------|
| `A` | AI 채팅 열기 (리스트/상세/비교 뷰) |
| `Ctrl+H` | 세션 기록 |
| `Enter` | 메시지 전송 |
| `Esc` | 채팅 닫기 / 스트림 취소 |
| `Ctrl+C` | 스트림 취소 |

## 확장 사고

어시스턴트는 복잡한 쿼리에 대해 확장 사고를 지원합니다. 활성화하면 최종 응답 전에 어시스턴트의 추론 과정을 보여주는 사고 인디케이터가 표시됩니다.

config.yaml에서 사고 예산을 설정합니다:
```yaml
ai:
  thinking_budget: 8000  # 확장 사고 최대 토큰 수 (기본값: 8000)
```

## 문제 해결

### "Bedrock not available in this region"

Bedrock은 모든 AWS 리전에서 사용할 수 있는 것은 아닙니다. 설정에서 지원되는 리전을 지정하십시오:

```yaml
ai:
  region: "us-west-2"  # Bedrock을 사용할 수 있는 리전 지정
```

### "Access Denied" 오류

IAM 역할/사용자에 필요한 Bedrock 권한이 있는지 확인하십시오. [IAM 권한](iam-permissions.ko.md#ai-채팅-선택-사항)을 참조하십시오.

### 도구 호출 제한 도달

"Tool call limit reached"가 표시되면 어시스턴트가 단일 쿼리에서 너무 많은 도구 호출을 수행한 것입니다. 제한을 늘리십시오:

```yaml
ai:
  max_tool_calls_per_query: 100  # 기본값 50에서 증가
```

### 세션이 유지되지 않는 경우

설정에서 세션 영속화를 활성화하십시오:

```yaml
ai:
  save_sessions: true  # 기본값: false
```

세션은 `~/.config/claws/chat/sessions/`에 저장됩니다.
