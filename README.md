# gstar-brief

GitHub에서 스타 마킹한 저장소를 수집하고 LLM으로 종합 브리핑 리포트를 생성하는 CLI 도구입니다.

## 설치

```bash
go install github.com/dp/gstar-brief@latest
```

또는 직접 빌드:

```bash
git clone https://github.com/dp/gstar-brief
cd gstar-brief
go build -o gstar-brief .
```

## 사용법

### 언어별 통계

```bash
gstar-brief stats
```

### 저장소 목록 조회

```bash
# 스타 카운트 순 정렬
gstar-brief list --sort stars

# 스타 마킹 시간 순 정렬
gstar-brief list --sort date

# 상위 N개만 출력
gstar-brief list --sort stars --limit 20
```

### LLM 브리핑 리포트 생성

```bash
gstar-brief report
```

### LLM API 호출 로그 확인

`--verbose` / `-v` 플래그를 사용하면 LLM API 요청/응답 상세 로그를 stderr로 출력합니다.

```bash
gstar-brief report --verbose
gstar-brief report -v
```

출력 예시:

```
time=2026-02-21T12:00:00 level=DEBUG msg="LLM API 요청" provider=claude model=claude-haiku-4-5 action=analyze target=owner/repo prompt_len=1240
time=2026-02-21T12:00:01 level=DEBUG msg="LLM API 응답" provider=claude model=claude-haiku-4-5 action=analyze target=owner/repo duration_ms=1230 input_tokens=450 output_tokens=180
```

로그 필드:

| 필드 | 설명 |
|---|---|
| `provider` | LLM provider 이름 (`claude`, `openai`, `gemini` 등) |
| `model` | 사용 중인 모델명 |
| `action` | `analyze` (저장소 분석) 또는 `report` (종합 리포트) |
| `target` | 분석 대상 저장소 (`owner/repo`), report 시 빈 값 |
| `prompt_len` | 프롬프트 바이트 수 |
| `duration_ms` | API 응답 소요 시간 (밀리초) |
| `input_tokens` | 입력 토큰 수 |
| `output_tokens` | 출력 토큰 수 |

### 대용량 스타 분할 처리

스타 수가 많아 GitHub API Rate Limit에 걸리는 경우 `--offset`과 `--limit`을 조합하여 구간별로 처리합니다.
GitHub API는 최신 순(스타 마킹 날짜 내림차순)으로 반환하므로 offset은 최신 별 기준으로 건너뜁니다.

```bash
# 1~500번째 (가장 최근에 스타한 저장소)
gstar-brief report --offset 0   --limit 500 --output report_1.md

# 501~1000번째
gstar-brief report --offset 500 --limit 500 --output report_2.md

# 1001~1500번째
gstar-brief report --offset 1000 --limit 500 --output report_3.md
```

통계도 동일하게 분할 처리할 수 있습니다.

```bash
gstar-brief stats --offset 0    --limit 500
gstar-brief stats --offset 500  --limit 500
gstar-brief stats --offset 1000 --limit 500
```

## 설정

설정 우선순위: **CLI 플래그 > `GSTAR_*` 환경변수 > 레거시 환경변수 > 설정 파일 > 기본값**

### 빠른 시작 — 설정 파일

```bash
# 설정 파일 초기화
gstar-brief init

# 편집
$EDITOR ~/.config/gstar-brief/config.toml
```

`~/.config/gstar-brief/config.toml`:

```toml
[github]
token = "ghp_xxxxxxxxxxxx"
user  = "myusername"

[llm]
provider   = "gemini"
gemini_key = "AIzaxxxxxxxxxxxxxxxxxx"
```

### 환경변수

`GSTAR_` 접두사 환경변수를 권장합니다. 레거시 변수(`GITHUB_TOKEN` 등)도 폴백으로 지원합니다.

| 변수 | 설명 |
|---|---|
| `GSTAR_GITHUB_TOKEN` | GitHub Personal Access Token |
| `GSTAR_GITHUB_USER` | 분석할 GitHub 유저명 |
| `GSTAR_LLM_PROVIDER` | `claude` / `openai` / `gemini` / `openrouter` / `ollama` |
| `GSTAR_LLM_MODEL` | 모델 오버라이드 |
| `GSTAR_LLM_GEMINI_KEY` | Gemini API Key |
| `GSTAR_LLM_ANTHROPIC_KEY` | Anthropic API Key |
| `GSTAR_LLM_OPENAI_KEY` | OpenAI API Key |
| `GSTAR_LLM_OPENROUTER_KEY` | OpenRouter API Key |
| `GSTAR_LLM_OLLAMA_MODEL` | Ollama 모델명 |
| `GSTAR_CONFIG_DIR` | 설정 디렉토리 경로 오버라이드 |

### 예시 — 환경변수 방식

```bash
export GSTAR_GITHUB_USER=myusername
export GSTAR_GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export GSTAR_LLM_PROVIDER=gemini
export GSTAR_LLM_GEMINI_KEY=AIzaxxxxxxxxxxxxxxxxxx

gstar-brief report
```

### 예시 — 특정 설정 파일 지정

```bash
gstar-brief --config ~/work/gstar.toml report
```

## LLM Provider

| Provider | 기본 모델 | 비고 |
|---|---|---|
| `claude` | `claude-haiku-4-5` | Anthropic Claude API |
| `openai` | `gpt-5-nano` | OpenAI API |
| `gemini` | `gemini-2.5-flash-lite` | Google Gemini API |
| `openrouter` | `openai/gpt-5-nano` | OpenAI 호환, 수백 개 모델 접근 가능 |
| `ollama` | (직접 설정) | 로컬 실행 모델 |

## GitHub Token 발급

1. GitHub → Settings → Developer settings → Personal access tokens
2. `public_repo` 스코프 선택 (비공개 저장소 분석 시 `repo` 스코프 필요)
3. 발급된 토큰을 `GITHUB_TOKEN` 환경변수에 설정
