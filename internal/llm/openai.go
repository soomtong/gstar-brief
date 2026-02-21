package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultOpenAIModel = "gpt-5-nano"

type openAIProvider struct {
	client *openai.Client
	model  string
}

func newOpenAI() (Provider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY 환경변수가 설정되지 않았습니다")
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &openAIProvider{
		client: &client,
		model:  modelOrDefault(defaultOpenAIModel),
	}, nil
}

func (p *openAIProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, analyzePrompt(repo))
}

func (p *openAIProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, reportPrompt(summaries))
}

func (p *openAIProvider) complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI API 응답이 비어있습니다")
	}

	return resp.Choices[0].Message.Content, nil
}
