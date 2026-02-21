package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// RepoContext는 LLM에 전달할 저장소 컨텍스트입니다.
type RepoContext struct {
	Name         string
	FullName     string
	Description  string
	Language     *string
	Stars        int
	StarredAt    time.Time
	Topics       []string
	Readme       string
	CodeSnippets []string
}

// RepoSummary는 저장소별 LLM 분석 결과입니다.
type RepoSummary struct {
	FullName  string
	Language  *string
	Stars     int
	StarredAt time.Time
	Summary   string
}

// Provider는 LLM provider 인터페이스입니다.
type Provider interface {
	// Analyze는 단일 저장소를 분석하여 요약을 반환합니다.
	Analyze(ctx context.Context, repo RepoContext) (string, error)
	// Report는 전체 요약 목록을 바탕으로 종합 브리핑 리포트를 생성합니다.
	Report(ctx context.Context, summaries []RepoSummary) (string, error)
}

// New는 환경변수 LLM_PROVIDER 기반으로 Provider를 생성합니다.
func New() (Provider, error) {
	providerName := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if providerName == "" {
		return nil, fmt.Errorf("LLM_PROVIDER 환경변수가 설정되지 않았습니다 (claude/openai/gemini/openrouter/ollama)")
	}

	switch providerName {
	case "claude":
		return newClaude()
	case "openai":
		return newOpenAI()
	case "gemini":
		return newGemini()
	case "openrouter":
		return newOpenRouter()
	case "ollama":
		return newOllama()
	default:
		return nil, fmt.Errorf("지원하지 않는 LLM_PROVIDER: %q (claude/openai/gemini/openrouter/ollama)", providerName)
	}
}

// modelOrDefault는 LLM_MODEL 환경변수 오버라이드 또는 기본 모델을 반환합니다.
func modelOrDefault(defaultModel string) string {
	if m := os.Getenv("LLM_MODEL"); m != "" {
		return m
	}
	return defaultModel
}

// requireEnvVars는 필수 환경변수들이 설정되어 있는지 검사합니다.
// 누락된 환경변수가 있으면 errors.Join으로 결합된 에러를 반환합니다.
func requireEnvVars(vars ...string) error {
	var errs []error
	for _, v := range vars {
		if os.Getenv(v) == "" {
			errs = append(errs, fmt.Errorf("%s 환경변수가 설정되지 않았습니다", v))
		}
	}
	return errors.Join(errs...)
}

// langOrDefault는 *string 언어 필드가 nil이면 기본값 포인터를 반환합니다.
// Go 1.26 new(expr): 기본값 문자열의 포인터를 한 줄로 표현합니다.
func langOrDefault(lang *string) *string {
	if lang != nil {
		return lang
	}
	return new("미분류")
}

// analyzePrompt는 저장소 분석 프롬프트를 생성합니다.
func analyzePrompt(repo RepoContext) string {
	var sb strings.Builder

	sb.WriteString("다음 GitHub 저장소를 분석하고 한국어로 2~3문장 요약을 작성해주세요.\n\n")
	sb.WriteString(fmt.Sprintf("저장소: %s\n", repo.FullName))
	if repo.Description != "" {
		sb.WriteString(fmt.Sprintf("설명: %s\n", repo.Description))
	}
	if repo.Language != nil {
		sb.WriteString(fmt.Sprintf("주언어: %s\n", *repo.Language))
	}
	sb.WriteString(fmt.Sprintf("스타: %d\n", repo.Stars))
	if len(repo.Topics) > 0 {
		sb.WriteString(fmt.Sprintf("토픽: %s\n", strings.Join(repo.Topics, ", ")))
	}

	if repo.Readme != "" {
		readme := repo.Readme[:min(len(repo.Readme), 3000)]
		if len(repo.Readme) > 3000 {
			readme += "\n...(생략)"
		}
		sb.WriteString(fmt.Sprintf("\n--- README ---\n%s\n", readme))
	}

	for i, snippet := range repo.CodeSnippets {
		if i >= 2 {
			break
		}
		code := snippet[:min(len(snippet), 2000)]
		if len(snippet) > 2000 {
			code += "\n...(생략)"
		}
		sb.WriteString(fmt.Sprintf("\n--- 코드 ---\n%s\n", code))
	}

	sb.WriteString("\n요약 (2~3문장, 한국어):")
	return sb.String()
}

// reportPrompt는 전체 브리핑 리포트 프롬프트를 생성합니다.
func reportPrompt(summaries []RepoSummary) string {
	var sb strings.Builder

	sb.WriteString("다음은 GitHub 스타 저장소 목록과 각 저장소의 요약입니다.\n")
	sb.WriteString("전체를 종합하여 한국어로 브리핑 리포트를 Markdown 형식으로 작성해주세요.\n")
	sb.WriteString("다음 항목을 포함해주세요:\n")
	sb.WriteString("1. 전체 요약 (주요 관심 분야, 기술 스택 트렌드)\n")
	sb.WriteString("2. 언어/분야별 분류\n")
	sb.WriteString("3. 특히 주목할 저장소 TOP 5 (이유 포함)\n\n")
	sb.WriteString("--- 저장소 목록 ---\n\n")

	for _, s := range summaries {
		lang := *langOrDefault(s.Language)
		sb.WriteString(fmt.Sprintf("### %s\n", s.FullName))
		sb.WriteString(fmt.Sprintf("- 언어: %s | 스타: %d | 스타 날짜: %s\n", lang, s.Stars, s.StarredAt.Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("- 요약: %s\n\n", s.Summary))
	}

	return sb.String()
}
