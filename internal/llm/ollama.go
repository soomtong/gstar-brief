package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultOllamaBaseURL = "http://localhost:11434/v1"

type ollamaProvider struct {
	client *openai.Client
	model  string
}

func newOllama() (Provider, error) {
	if err := requireEnvVars("OLLAMA_MODEL"); err != nil {
		return nil, err
	}
	model := os.Getenv("OLLAMA_MODEL")

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	} else {
		// OLLAMA_BASE_URL이 /v1 없이 설정된 경우 보정
		baseURL = strings.TrimRight(baseURL, "/") + "/v1"
	}

	client := openai.NewClient(
		option.WithAPIKey("ollama"), // Ollama는 API 키 불필요, 더미값 사용
		option.WithBaseURL(baseURL),
	)

	return &ollamaProvider{
		client: &client,
		model:  model,
	}, nil
}

func (p *ollamaProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, "analyze", repo.FullName, analyzePrompt(repo))
}

func (p *ollamaProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, "report", "", reportPrompt(summaries))
}

func (p *ollamaProvider) complete(ctx context.Context, action, target, prompt string) (string, error) {
	slog.Debug("LLM API 요청",
		"provider", "ollama",
		"model", p.model,
		"action", action,
		"target", target,
		"prompt_len", len(prompt),
	)

	start := time.Now()
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	elapsed := time.Since(start)

	if err != nil {
		slog.Debug("LLM API 오류",
			"provider", "ollama",
			"model", p.model,
			"action", action,
			"target", target,
			"duration_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("Ollama API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Ollama API 응답이 비어있습니다")
	}

	slog.Debug("LLM API 응답",
		"provider", "ollama",
		"model", p.model,
		"action", action,
		"target", target,
		"duration_ms", elapsed.Milliseconds(),
		"input_tokens", resp.Usage.PromptTokens,
		"output_tokens", resp.Usage.CompletionTokens,
	)

	return resp.Choices[0].Message.Content, nil
}
