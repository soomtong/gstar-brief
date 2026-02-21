package llm

import (
	"context"
	"fmt"
	"os"

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
	return p.complete(ctx, analyzePrompt(repo))
}

func (p *claudeProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, reportPrompt(summaries))
}

func (p *claudeProvider) complete(ctx context.Context, prompt string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Claude API 호출 실패: %w", err)
	}

	if len(msg.Content) == 0 {
		return "", fmt.Errorf("Claude API 응답이 비어있습니다")
	}

	return msg.Content[0].Text, nil
}
