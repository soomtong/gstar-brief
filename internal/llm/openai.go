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

const defaultOpenAIModel = "gpt-5-nano"

type openAIProvider struct {
	client *openai.Client
	model  string
}

func newOpenAI() (Provider, error) {
	if err := requireEnvVars("OPENAI_API_KEY"); err != nil {
		return nil, err
	}
	client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
	return &openAIProvider{
		client: &client,
		model:  modelOrDefault(defaultOpenAIModel),
	}, nil
}

func (p *openAIProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, "analyze", repo.FullName, analyzePrompt(repo))
}

func (p *openAIProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, "report", "", reportPrompt(summaries))
}

func (p *openAIProvider) complete(ctx context.Context, action, target, prompt string) (string, error) {
	slog.Debug("LLM API 요청",
		"provider", "openai",
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
			"provider", "openai",
			"model", p.model,
			"action", action,
			"target", target,
			"duration_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("OpenAI API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI API 응답이 비어있습니다")
	}

	slog.Debug("LLM API 응답",
		"provider", "openai",
		"model", p.model,
		"action", action,
		"target", target,
		"duration_ms", elapsed.Milliseconds(),
		"input_tokens", resp.Usage.PromptTokens,
		"output_tokens", resp.Usage.CompletionTokens,
	)

	return resp.Choices[0].Message.Content, nil
}
