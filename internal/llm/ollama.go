package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultOllamaBaseURL = "http://localhost:11434/v1"

type ollamaProvider struct {
	client *openai.Client
	model  string
}

func newOllama() (Provider, error) {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		return nil, fmt.Errorf("OLLAMA_MODEL 환경변수가 설정되지 않았습니다")
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	} else {
		// OLLAMA_BASE_URL이 /v1 없이 설정된 경우 보정
		if len(baseURL) > 0 && baseURL[len(baseURL)-1] != '/' {
			baseURL += "/v1"
		}
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
	return p.complete(ctx, analyzePrompt(repo))
}

func (p *ollamaProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, reportPrompt(summaries))
}

func (p *ollamaProvider) complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Ollama API 호출 실패: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Ollama API 응답이 비어있습니다")
	}

	return resp.Choices[0].Message.Content, nil
}
