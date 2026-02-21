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

## Go 표준 라이브러리 현대화 가이드

개발 시 Go 1.20 이상의 표준 라이브러리와 1.26 신기능을 적극 활용합니다.

### `errors.Join` (Go 1.20) — 다중 에러 결합

여러 필수 환경변수 누락을 한 번에 보고할 때 사용합니다. `requireEnvVars` 헬퍼가 이 패턴을 캡슐화합니다.

```go
// internal/llm/provider.go
func requireEnvVars(vars ...string) error {
    var errs []error
    for _, v := range vars {
        if os.Getenv(v) == "" {
            errs = append(errs, fmt.Errorf("%s 환경변수가 설정되지 않았습니다", v))
        }
    }
    return errors.Join(errs...)
}

// 각 provider 초기화에서 사용
func newClaude() (Provider, error) {
    if err := requireEnvVars("ANTHROPIC_API_KEY"); err != nil {
        return nil, err
    }
    // ...
}
```

### `slices.SortFunc` (Go 1.21) — 타입 안전 정렬

`sort.Slice`의 클로저 방식 대신 타입 파라미터를 활용한 정렬을 사용합니다.

```go
// 이전 방식
sort.Slice(repos, func(i, j int) bool {
    return repos[i].StargazersCount > repos[j].StargazersCount
})

// Go 1.21 방식 — 비교 함수가 타입 안전
slices.SortFunc(repos, func(a, b github.Repo) int {
    return b.StargazersCount - a.StargazersCount
})

// time.Time 비교: Compare() 메서드 활용
slices.SortFunc(repos, func(a, b github.Repo) int {
    return b.StarredAt.Compare(a.StarredAt)
})
```

### `maps.Keys` + `slices.Collect` (Go 1.21) — 맵 키 추출

맵을 순회하여 슬라이스로 변환할 때 사용합니다.

```go
// cmd/stats.go — 언어별 통계 정렬
langs := slices.Collect(maps.Keys(langCount))
stats := make([]stat, 0, len(langs))
for _, lang := range langs {
    stats = append(stats, stat{lang, langCount[lang], langStars[lang]})
}
slices.SortFunc(stats, func(a, b stat) int { return b.count - a.count })
```

### `min` / `max` 내장 함수 (Go 1.21) — 값 범위 제한

임시 변수 없이 최소/최대값을 표현합니다.

```go
// 문자열 길이 제한 (provider.go)
readme := repo.Readme[:min(len(repo.Readme), 3000)]

// 막대 그래프 최소 길이 보장 (stats.go)
bar := strings.Repeat("█", max(barLen, 1))
```

### `log/slog` (Go 1.21) — 구조화 로깅

`fmt.Printf` 진행 로그 대신 키-값 쌍 구조화 로깅을 사용합니다. 라이브러리 코드(`internal/`)에서는 기본 slog 로거를 사용하여 호출자가 핸들러를 교체할 수 있도록 합니다.

```go
// 이전 방식
fmt.Printf("  [%d/%d] %s\n", idx+1, len(repos), repo.FullName)

// Go 1.21 방식 — 속성이 구조화됨
slog.Info("저장소 분석 완료", "repo", repo.FullName, "idx", idx+1, "total", len(repos))
slog.Warn("저장소 분석 실패", "error", err)
```

### `iter.Seq2` + `range over func` (Go 1.23) — 이터레이터

페이지네이션처럼 반복 로직을 재사용 가능한 이터레이터로 추상화합니다. `iter.Seq2[V, E]`는 값과 에러를 함께 yield하는 표준 패턴입니다.

```go
// internal/github/client.go — 페이지네이션 이터레이터
func (c *Client) starredPages(ctx context.Context, username string) iter.Seq2[starredItem, error] {
    return func(yield func(starredItem, error) bool) {
        for page := 1; ; page++ {
            // ... HTTP 요청 ...
            for _, item := range items {
                if !yield(item, nil) {
                    return  // 소비자가 break하면 자동 종료
                }
            }
            if len(items) < perPage { return }
        }
    }
}

// 소비자: range over func으로 자연스럽게 순회
func (c *Client) ListStarred(ctx context.Context, username string, limit int) ([]Repo, error) {
    var repos []Repo
    for item, err := range c.starredPages(ctx, username) {
        if err != nil { return nil, err }
        repos = append(repos, item.Repo)
        if limit > 0 && len(repos) >= limit { break }
    }
    return repos, nil
}
```

### `os.WriteFile` (Go 1.16) — 파일 쓰기 단순화

`os.Create` + `defer Close` + `fmt.Fprintln` 3단계를 한 줄로 대체합니다.

```go
// 이전 방식
f, err := os.Create(path)
if err != nil { return err }
defer f.Close()
_, err = fmt.Fprintln(f, content)

// Go 1.16 방식
err = os.WriteFile(path, []byte(content+"\n"), 0o644)
```

### `new(expr)` (Go 1.26) — 초기값 있는 포인터 생성

`new` 내장 함수에 초기값 표현식을 직접 전달할 수 있습니다. nil 가능 필드의 기본값 포인터를 한 줄로 표현합니다.

```go
// 이전 방식 (Go 1.25 이하)
defaultLang := "미분류"
return &defaultLang

// Go 1.26 방식
return new("미분류")

// internal/llm/provider.go 실제 적용 예
func langOrDefault(lang *string) *string {
    if lang != nil { return lang }
    return new("미분류")
}
```

### Green Tea GC (Go 1.26)

Go 1.26에서 Green Tea GC가 기본 활성화됩니다. 소형 객체의 마킹/스캔 성능이 10~40% 향상되며, 별도 설정 없이 자동 적용됩니다.

```bash
# 비활성화 (문제 발생 시에만)
GOEXPERIMENT=nogreenteagc go build .
```

### `go fix` 현대화 도구 (Go 1.26)

Go 1.26의 `go fix`는 완전히 재작성되어 코드베이스를 최신 관용구로 자동 업데이트합니다. 개발 이터레이션 중 주기적으로 실행합니다.

```bash
# 전체 모듈 현대화 적용
go fix ./...

# 적용 가능한 fix 목록 확인
go vet ./...
```

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
    Name         string
    FullName     string
    Description  string
    Language     *string  // nil이면 언어 미분류
    Stars        int
    StarredAt    time.Time
    Topics       []string
    Readme       string
    CodeSnippets []string
}

type RepoSummary struct {
    FullName  string
    Language  *string  // nil이면 언어 미분류
    Stars     int
    StarredAt time.Time
    Summary   string
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
- 기본 모델: `gemini-2.5-flash-lite`
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
- 기본 엔드포인트: `http://localhost:11434/v1` (`OLLAMA_BASE_URL`로 오버라이드)
- `OLLAMA_BASE_URL` 설정 시 끝 슬래시 유무와 무관하게 `/v1` 자동 보정: `strings.TrimRight(baseURL, "/") + "/v1"`
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

## 설정 시스템

### 우선순위 (높음 → 낮음)

```
1. CLI 플래그          --limit 10 --output report.md
2. GSTAR_* 환경변수    GSTAR_GITHUB_USER=dp GSTAR_LLM_PROVIDER=gemini
3. 레거시 환경변수     GITHUB_TOKEN, LLM_PROVIDER 등 (폴백)
4. 설정 파일           ~/.config/gstar-brief/config.toml
5. 기본값              ollama_base_url = "http://localhost:11434"
```

### 설정 파일

- **경로**: `~/.config/gstar-brief/config.toml`
- **포맷**: TOML
- **생성**: `gstar-brief init`
- **경로 오버라이드**: `GSTAR_CONFIG_DIR=/path/to/dir` 또는 `--config /path/to/config.toml`
- `XDG_CONFIG_HOME` 환경변수 존중 (Linux/macOS 모두)

```toml
[github]
token = "ghp_..."
user  = "dp"

[llm]
provider      = "gemini"
model         = ""           # 비워두면 provider 기본값 사용
gemini_key    = "AIza..."
```

### 환경변수

`GSTAR_` 접두사가 붙은 변수가 레거시 변수보다 우선합니다.

| GSTAR_* 변수 | 레거시 변수 | 설명 |
|---|---|---|
| `GSTAR_GITHUB_TOKEN` | `GITHUB_TOKEN` | GitHub PAT | 
| `GSTAR_GITHUB_USER` | `GITHUB_USER` | 분석할 GitHub 유저명 |
| `GSTAR_LLM_PROVIDER` | `LLM_PROVIDER` | `claude` / `openai` / `gemini` / `openrouter` / `ollama` |
| `GSTAR_LLM_MODEL` | `LLM_MODEL` | 모델 오버라이드 |
| `GSTAR_LLM_ANTHROPIC_KEY` | `ANTHROPIC_API_KEY` | Claude 사용 시 |
| `GSTAR_LLM_OPENAI_KEY` | `OPENAI_API_KEY` | OpenAI 사용 시 |
| `GSTAR_LLM_GEMINI_KEY` | `GEMINI_API_KEY` | Gemini 사용 시 |
| `GSTAR_LLM_OPENROUTER_KEY` | `OPENROUTER_API_KEY` | OpenRouter 사용 시 |
| `GSTAR_LLM_OPENROUTER_MODEL` | `OPENROUTER_MODEL` | OpenRouter 모델 ID |
| `GSTAR_LLM_OLLAMA_BASE_URL` | `OLLAMA_BASE_URL` | Ollama 엔드포인트 |
| `GSTAR_LLM_OLLAMA_MODEL` | `OLLAMA_MODEL` | Ollama 모델명 |
| `GSTAR_CONFIG_DIR` | — | 설정 디렉토리 경로 오버라이드 |

### 구현 패턴 (`internal/config/config.go`)

- `config.Init(cfgFile)` — cobra `OnInitialize`에서 호출
- `config.Load()` — `Config` 구조체로 언마샬
- `cfg.ApplyToEnv()` — SDK가 읽는 표준 환경변수에 설정값 적용
- `config.DefaultTOML()` — `init` 커맨드용 템플릿 반환

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
