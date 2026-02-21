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

### 환경변수

| 변수 | 설명 | 필수 |
|---|---|---|
| `GITHUB_TOKEN` | GitHub Personal Access Token | 권장 |
| `GITHUB_USER` | 분석할 GitHub 유저명 | 필수 |
| `LLM_PROVIDER` | 사용할 LLM provider (`claude` / `openai` / `gemini` / `openrouter` / `ollama`) | 필수 |
| `LLM_MODEL` | 모델 오버라이드 (미설정 시 provider 기본값 사용) | 선택 |
| `ANTHROPIC_API_KEY` | Claude 사용 시 | 조건부 |
| `OPENAI_API_KEY` | OpenAI 사용 시 | 조건부 |
| `GEMINI_API_KEY` | Gemini 사용 시 | 조건부 |
| `OPENROUTER_API_KEY` | OpenRouter 사용 시 | 조건부 |
| `OPENROUTER_MODEL` | OpenRouter 모델 ID (기본: `openai/gpt-5-nano`) | 선택 |
| `OLLAMA_BASE_URL` | Ollama 엔드포인트 (기본: `http://localhost:11434`) | 조건부 |

### 예시

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export GITHUB_USER=myusername
export LLM_PROVIDER=gemini
export GEMINI_API_KEY=AIzaxxxxxxxxxxxxxxxxxx

gstar-brief report
```

## LLM Provider

| Provider | 기본 모델 | 비고 |
|---|---|---|
| `claude` | `claude-haiku-4-5` | Anthropic Claude API |
| `openai` | `gpt-5-nano` | OpenAI API |
| `gemini` | `gemini-2.0-flash` | Google Gemini API |
| `openrouter` | `openai/gpt-5-nano` | OpenAI 호환, 수백 개 모델 접근 가능 |
| `ollama` | (직접 설정) | 로컬 실행 모델 |

## GitHub Token 발급

1. GitHub → Settings → Developer settings → Personal access tokens
2. `public_repo` 스코프 선택 (비공개 저장소 분석 시 `repo` 스코프 필요)
3. 발급된 토큰을 `GITHUB_TOKEN` 환경변수에 설정
