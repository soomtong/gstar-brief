package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	defaultOpenRouterModel = "openai/gpt-5-nano"
	openRouterBaseURL      = "https://openrouter.ai/api/v1"
)

type openRouterProvider struct {
	client *openai.Client
	model  string
}

func newOpenRouter() (Provider, error) {
	if err := requireEnvVars("OPENROUTER_API_KEY"); err != nil {
		return nil, err
	}

	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = modelOrDefault(defaultOpenRouterModel)
	}

	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		option.WithBaseURL(openRouterBaseURL),
		option.WithHeader("HTTP-Referer", "https://github.com/dp/gstar-brief"),
		option.WithHeader("X-Title", "gstar-brief"),
	)

	return &openRouterProvider{
		client: &client,
		model:  model,
	}, nil
}

func (p *openRouterProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, "analyze", repo.FullName, analyzePrompt(repo))
}

func (p *openRouterProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, "report", "", reportPrompt(summaries))
}

func (p *openRouterProvider) complete(ctx context.Context, action, target, prompt string) (string, error) {
	slog.Debug("LLM API 요청",
		"provider", "openrouter",
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
			"provider", "openrouter",
			"model", p.model,
			"action", action,
			"target", target,
			"duration_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("OpenRouter API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenRouter API 응답이 비어있습니다")
	}

	slog.Debug("LLM API 응답",
		"provider", "openrouter",
		"model", p.model,
		"action", action,
		"target", target,
		"duration_ms", elapsed.Milliseconds(),
		"input_tokens", resp.Usage.PromptTokens,
		"output_tokens", resp.Usage.CompletionTokens,
	)

	return resp.Choices[0].Message.Content, nil
}
