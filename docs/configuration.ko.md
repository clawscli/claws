# 설정

## AWS 자격 증명

claws는 표준 AWS 설정을 사용합니다:

- `~/.aws/credentials` - AWS 자격 증명
- `~/.aws/config` - AWS 설정 (리전, 프로필)
- 환경 변수: `AWS_PROFILE`, `AWS_REGION`, `AWS_ACCESS_KEY_ID` 등

## 설정 파일

선택적 설정은 `~/.config/claws/config.yaml`에 저장할 수 있습니다.

### 사용자 지정 설정 파일 경로

기본값 대신 사용자 지정 설정 파일을 사용할 수 있습니다:

```bash
# CLI 플래그로 지정
claws -c /path/to/config.yaml
claws --config ~/work/claws-work.yaml

# 환경 변수로 지정
CLAWS_CONFIG=/path/to/config.yaml claws
```

**우선순위:** `-c` 플래그 > `CLAWS_CONFIG` 환경 변수 > 기본값 (`~/.config/claws/config.yaml`)

사용 예시:
- 환경별 설정 (업무용/개인용)
- 프로젝트별 설정을 사용한 CI/CD
- 다양한 설정으로 테스트

### 설정 파일 형식

```yaml
timeouts:
  aws_init: 10s           # AWS 초기화 타임아웃 (기본값: 5초)
  multi_region_fetch: 60s # 멀티 리전 병렬 가져오기 타임아웃 (기본값: 30초)
  tag_search: 45s         # 태그 검색 타임아웃 (기본값: 30초)
  metrics_load: 30s       # CloudWatch 메트릭 로드 타임아웃 (기본값: 30초)
  log_fetch: 15s          # CloudWatch Logs 가져오기 타임아웃 (기본값: 10초)

concurrency:
  max_fetches: 100        # 최대 동시 API 가져오기 수 (기본값: 50)

cloudwatch:
  window: 15m             # 메트릭 데이터 윈도우 기간 (기본값: 15m)

autosave:
  enabled: true           # 리전/프로필/테마/compact_header 변경 시 저장 (기본값: false)

compact_header: false     # 단일 행 컴팩트 헤더 사용 (기본값: false)

startup:                  # 시작 시 적용 (설정이 있는 경우)
  view: services          # 시작 뷰: "dashboard", "services" 또는 "service/resource" (예: "ec2", "rds/snapshots")
  filter: bastion         # 시작 시 적용할 퍼지 필터 (`/`와 동일). CLI -f/--filter가 우선
  tag: Role=bastion       # 시작 시 적용할 태그 필터 (`:tag`와 동일). CLI --tag가 우선
  profiles:               # 다중 프로필 지원
    - production
  regions:
    - us-east-1
    - us-west-2

navigation:
  max_stack_size: 100     # 탐색 기록 최대 깊이 (기본값: 100)

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

theme: nord               # 프리셋: dark, light, nord, dracula, gruvbox, catppuccin

# 프리셋에 사용자 지정 오버라이드를 적용하는 경우:
# theme:
#   preset: dracula
#   primary: "#ff79c6"
#   danger: "#ff5555"
```

설정 파일은 **자동으로 생성되지 않습니다**. 필요한 경우 수동으로 생성하십시오.

CLI 플래그(`-p`, `-r`, `-t`, `--compact`, `--no-compact`, `--autosave`, `--no-autosave`)는 설정 파일의 값을 덮어씁니다.
여러 값을 지정할 수 있습니다: `-p dev,prod` 또는 `-p dev -p prod`.

### 특수 프로필 ID

| ID | 설명 | 동등한 동작 |
|----|------|-------------|
| `__sdk_default__` | AWS SDK 기본 자격 증명 체인 사용 | (`-p` 플래그 없음) |
| `__env_only__` | ~/.aws를 무시하고 환경 변수/IMDS/ECS/Lambda 자격 증명만 사용 | `-e` 플래그 |

```bash
# -p 플래그로 환경 변수 전용 모드 사용
claws -p __env_only__

# 명명된 프로필과 특수 모드를 조합 (둘 다 쿼리)
claws -p production,__env_only__
```

이러한 ID는 `startup.profiles`에서도 사용할 수 있습니다:

```yaml
startup:
  profiles:
    - __sdk_default__
    - production
```


## 테마

claws에는 6개의 내장 색상 테마가 포함되어 있습니다:

| 테마 | 설명 |
|------|------|
| `dark` | 기본 다크 테마 (핑크/마젠타 강조) |
| `light` | 밝은 배경 터미널용 |
| `nord` | 북유럽풍 차분한 블루 팔레트 |
| `dracula` | 인기 다크 테마 (퍼플/핑크) |
| `gruvbox` | 레트로 따뜻한 어스 톤 |
| `catppuccin` | 모던 파스텔 컬러 (Mocha 변형) |

### 테마 미리보기

| dark | light | nord |
|------|-------|------|
| ![dark](images/theme-dark.png) | ![light](images/theme-light.png) | ![nord](images/theme-nord.png) |

| dracula | gruvbox | catppuccin |
|---------|---------|------------|
| ![dracula](images/theme-dracula.png) | ![gruvbox](images/theme-gruvbox.png) | ![catppuccin](images/theme-catppuccin.png) |

### 테마 전환

```bash
# 커맨드 라인으로 지정
claws -t nord

# 커맨드 모드로 전환 (런타임)
:theme dracula
```

autosave가 활성화된 경우, 테마 변경 사항이 설정 파일에 저장됩니다.

### 사용자 지정 테마 색상

프리셋의 특정 색상을 오버라이드할 수 있습니다:

```yaml
theme:
  preset: dracula
  primary: "#ff79c6"
  danger: "#ff5555"
  success: "#50fa7b"
```

## 읽기 전용 모드

모든 파괴적 액션을 비활성화합니다:

```bash
# 플래그로 지정
claws --read-only

# 환경 변수로 지정
CLAWS_READ_ONLY=1 claws
```

## 디버그 로깅

파일에 디버그 로그를 활성화합니다:

```bash
claws -l debug.log
```

## IAM 권한

필요한 IAM 권한에 대해서는 [IAM 권한](iam-permissions.ko.md)을 참조하십시오.
