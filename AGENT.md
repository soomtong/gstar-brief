# AGENT.md

Claude Code 에이전트를 위한 개발 가이드입니다.

## 프로젝트 목표

GitHub에서 스타 마킹한 저장소 목록을 수집하고, 각 저장소의 README와 주요 코드를 LLM으로 분석하여 종합 브리핑 리포트를 생성하는 Go CLI 도구입니다.

## 기술 스택

| 구성요소 | 선택 |
|---|---|
| 언어 | Go 1.26+ |
| GitHub 데이터 수집 | GitHub REST API v3 |
| LLM 연동 | 추상화된 provider 인터페이스 |
| CLI 프레임워크 | cobra |

## Go 1.26 현대화 가이드

개발 시 Go 1.26의 신기능을 적극 활용합니다.

### `new(expr)` - 초기값 있는 포인터 생성

Go 1.26부터 `new` 내장 함수에 초기값 표현식을 직접 전달할 수 있습니다. 포인터 필드를 가진 구조체 초기화 시 임시 변수 없이 간결하게 작성합니다.

```go
// 이전 방식 (Go 1.25 이하)
stars := 42
repo := Repo{Stars: &stars}

// Go 1.26 방식
repo := Repo{Stars: new(42)}

// JSON 직렬화 등 optional 필드에 특히 유용
type RepoContext struct {
    Stars    *int    // nil이면 정보 없음
    Language *string // nil이면 언어 미분류
}

ctx := RepoContext{
    Stars:    new(repo.StargazersCount),
    Language: new(repo.Language),
}
```

### Green Tea GC

Go 1.26에서 Green Tea GC가 기본 활성화됩니다. 소형 객체의 마킹/스캔 성능이 10~40% 향상되며, 별도 설정 없이 자동 적용됩니다. 비활성화가 필요한 경우에만 빌드 시 명시합니다.

```bash
# 비활성화 (문제 발생 시에만)
GOEXPERIMENT=nogreenteagc go build .
```

### `go fix` 현대화 도구

Go 1.26의 `go fix`는 완전히 재작성되어 코드베이스를 최신 관용구로 자동 업데이트합니다. 개발 이터레이션 중 주기적으로 실행합니다.

```bash
# 전체 모듈 현대화 적용
go fix ./...

# 특정 패키지만
go fix ./internal/llm/...

# 적용 가능한 fix 목록 확인 (vet 방식으로 진단)
go vet ./...
```

`go fix`는 프로그램 동작을 변경하지 않는 안전한 변환만 수행합니다. CI 파이프라인 또는 코드 리뷰 전에 실행하여 항상 최신 Go 관용구를 유지합니다.

## 디렉토리 구조

```
gstar-brief/
├── cmd/
│   ├── root.go        # 루트 커맨드 및 전역 플래그
│   ├── list.go        # gstar-brief list
│   ├── stats.go       # gstar-brief stats
│   └── report.go      # gstar-brief report
├── internal/
│   ├── github/
│   │   └── client.go  # GitHub REST API 클라이언트
│   ├── llm/
│   │   ├── provider.go     # Provider 인터페이스 정의
│   │   ├── claude.go       # Anthropic Claude
│   │   ├── openai.go       # OpenAI
│   │   ├── gemini.go       # Google Gemini
│   │   ├── openrouter.go   # OpenRouter (OpenAI 호환)
│   │   └── ollama.go       # 로컬 Ollama
│   ├── analyzer/
│   │   └── analyzer.go    # 저장소 분석 파이프라인
│   └── report/
│       └── report.go      # 브리핑 리포트 생성 및 출력
├── main.go
├── go.mod
├── README.md
└── AGENT.md
```

## GitHub REST API

### 스타 저장소 목록 조회

```
GET /users/{username}/starred
```

- 페이지네이션 필수: `?per_page=100&page=N`
- `starred_at` 필드를 포함하려면 헤더 필요:
  ```
  Accept: application/vnd.github.star+json
  ```
- 응답 필드: `repo.name`, `repo.full_name`, `repo.description`, `repo.language`, `repo.stargazers_count`, `starred_at`

### README 조회

```
GET /repos/{owner}/{repo}/readme
```

- 응답의 `content` 필드는 base64 인코딩 → 디코딩 필요
- `encoding` 필드로 인코딩 방식 확인

### 주요 코드 파일 조회

```
GET /repos/{owner}/{repo}/contents/{path}
```

- 분석 대상 파일 우선순위: `main.go`, `index.ts`, `app.py`, `src/` 디렉토리 내 주요 파일
- 파일이 너무 크면 LLM 컨텍스트 초과 주의 → 최대 10KB 내외로 제한

### Rate Limit

- 인증(Token 사용) 시: 5,000 req/h
- 미인증 시: 60 req/h
- `GITHUB_TOKEN` 설정을 강력히 권장
- 응답 헤더 `X-RateLimit-Remaining` 으로 잔여 횟수 확인

## LLM Provider 인터페이스

```go
type Provider interface {
    Analyze(ctx context.Context, repo RepoContext) (string, error)
    Report(ctx context.Context, summaries []RepoSummary) (string, error)
}

type RepoContext struct {
    Name        string
    FullName    string
    Description string
    Language    string
    Stars       int
    StarredAt   time.Time
    Readme      string
    CodeSnippets []string
}

type RepoSummary struct {
    FullName string
    Summary  string
}
```

## LLM Provider 구현

### Claude (`internal/llm/claude.go`)

- SDK: `github.com/anthropics/anthropic-sdk-go`
- 기본 모델: `claude-haiku-4-5`
- 인증: `ANTHROPIC_API_KEY`

### OpenAI (`internal/llm/openai.go`)

- SDK: `github.com/openai/openai-go`
- 기본 모델: `gpt-5-nano`
- 인증: `OPENAI_API_KEY`

### Gemini (`internal/llm/gemini.go`)

- SDK: `google.golang.org/genai`
- 기본 모델: `gemini-2.0-flash`
- 인증: `GEMINI_API_KEY`
- 엔드포인트: `generativelanguage.googleapis.com`

### OpenRouter (`internal/llm/openrouter.go`)

- OpenAI 호환 API → OpenAI SDK 재사용 가능
- 기본 엔드포인트: `https://openrouter.ai/api/v1`
- 기본 모델: `openai/gpt-5-nano` (모델 ID 형식: `provider/model-name`)
- 인증: `OPENROUTER_API_KEY`
- 선택적 헤더: `HTTP-Referer`, `X-Title`
- 환경변수 `OPENROUTER_MODEL`로 모델 오버라이드 가능

### Ollama (`internal/llm/ollama.go`)

- OpenAI 호환 API → OpenAI SDK 재사용 가능
- 기본 엔드포인트: `http://localhost:11434` (`OLLAMA_BASE_URL`로 오버라이드)
- 모델: `OLLAMA_MODEL` 환경변수로 설정 (필수)

## 분석 파이프라인

```
starred repos 수집 (GitHub API, 페이지네이션)
    ↓
각 저장소별:
  1. README 조회 및 디코딩
  2. 주요 코드 파일 조회 (최대 3개 파일, 각 10KB 이내)
  3. LLM.Analyze() 호출 → 저장소 요약 생성
    ↓
전체 요약 목록을 LLM.Report() 에 전달
    ↓
종합 브리핑 리포트 출력 (stdout 또는 파일)
```

## 환경변수

| 변수 | 설명 | 필수 |
|---|---|---|
| `GITHUB_TOKEN` | GitHub Personal Access Token | 권장 |
| `GITHUB_USER` | 분석할 GitHub 유저명 | 필수 |
| `LLM_PROVIDER` | `claude` / `openai` / `gemini` / `openrouter` / `ollama` | 필수 |
| `LLM_MODEL` | provider 기본 모델 오버라이드 | 선택 |
| `ANTHROPIC_API_KEY` | Claude 사용 시 | 조건부 |
| `OPENAI_API_KEY` | OpenAI 사용 시 | 조건부 |
| `GEMINI_API_KEY` | Gemini 사용 시 | 조건부 |
| `OPENROUTER_API_KEY` | OpenRouter 사용 시 | 조건부 |
| `OPENROUTER_MODEL` | OpenRouter 모델 ID (기본: `openai/gpt-5-nano`) | 선택 |
| `OLLAMA_BASE_URL` | Ollama 엔드포인트 (기본: `http://localhost:11434`) | 조건부 |
| `OLLAMA_MODEL` | Ollama 모델명 | 조건부 |

## 빌드 및 테스트

```bash
# 빌드
go build -o gstar-brief .

# 테스트
go test ./...

# 특정 패키지 테스트
go test ./internal/github/...
go test ./internal/llm/...

# 린트 (golangci-lint 필요)
golangci-lint run

# 코드 현대화 (개발 이터레이션 중 주기적으로 실행)
go fix ./...
```

## 주의사항

- GitHub API 호출 시 저장소 수가 많으면 Rate Limit 도달 가능 → 요청 간 적절한 delay 추가
- LLM에 전달하는 코드 컨텍스트는 토큰 한도를 고려하여 크기 제한 필수
- `starred_at` 필드는 `Accept: application/vnd.github.star+json` 헤더 없이는 응답에 포함되지 않음
