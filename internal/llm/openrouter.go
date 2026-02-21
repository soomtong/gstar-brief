package llm

import (
	"context"
	"fmt"
	"os"

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
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY 환경변수가 설정되지 않았습니다")
	}

	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = modelOrDefault(defaultOpenRouterModel)
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
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
	return p.complete(ctx, analyzePrompt(repo))
}

func (p *openRouterProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, reportPrompt(summaries))
}

func (p *openRouterProvider) complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenRouter API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenRouter API 응답이 비어있습니다")
	}

	return resp.Choices[0].Message.Content, nil
}
