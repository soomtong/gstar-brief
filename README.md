# gstar-brief

GitHub에서 스타 마킹한 저장소를 수집하고 LLM으로 종합 브리핑 리포트를 생성하는 CLI 도구입니다.

## 주요 기능

- **언어별 통계** - 스타 저장소를 프로그래밍 언어별로 집계
- **스타 카운트 정렬** - 스타 수 기준으로 저장소 목록 정렬
- **시간순 정렬** - 스타 마킹한 시간 기준으로 저장소 목록 정렬
- **LLM 브리핑 리포트** - 각 저장소의 README와 주요 코드를 LLM이 분석하여 종합 리포트 생성
- **다양한 LLM Provider 지원** - Claude, OpenAI, Gemini, OpenRouter, Ollama

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
