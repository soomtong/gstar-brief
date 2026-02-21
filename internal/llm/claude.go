package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultClaudeModel = "claude-haiku-4-5"

type claudeProvider struct {
	client *anthropic.Client
	model  string
}

func newClaude() (Provider, error) {
	if err := requireEnvVars("ANTHROPIC_API_KEY"); err != nil {
		return nil, err
	}
	client := anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	return &claudeProvider{
		client: &client,
		model:  modelOrDefault(defaultClaudeModel),
	}, nil
}

func (p *claudeProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, "analyze", repo.FullName, analyzePrompt(repo))
}

func (p *claudeProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, "report", "", reportPrompt(summaries))
}

func (p *claudeProvider) complete(ctx context.Context, action, target, prompt string) (string, error) {
	slog.Debug("LLM API 요청",
		"provider", "claude",
		"model", p.model,
		"action", action,
		"target", target,
		"prompt_len", len(prompt),
	)

	start := time.Now()
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	elapsed := time.Since(start)

	if err != nil {
		slog.Debug("LLM API 오류",
			"provider", "claude",
			"model", p.model,
			"action", action,
			"target", target,
			"duration_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("Claude API 호출 실패: %w", err)
	}

	if len(msg.Content) == 0 {
		return "", fmt.Errorf("Claude API 응답이 비어있습니다")
	}

	slog.Debug("LLM API 응답",
		"provider", "claude",
		"model", p.model,
		"action", action,
		"target", target,
		"duration_ms", elapsed.Milliseconds(),
		"input_tokens", msg.Usage.InputTokens,
		"output_tokens", msg.Usage.OutputTokens,
	)

	return msg.Content[0].Text, nil
}
