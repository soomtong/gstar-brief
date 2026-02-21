package llm

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

const defaultGeminiModel = "gemini-2.5-flash-lite"

type geminiProvider struct {
	client *genai.Client
	model  string
}

func newGemini() (Provider, error) {
	if err := requireEnvVars("GEMINI_API_KEY"); err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("Gemini 클라이언트 생성 실패: %w", err)
	}

	return &geminiProvider{
		client: client,
		model:  modelOrDefault(defaultGeminiModel),
	}, nil
}

func (p *geminiProvider) Analyze(ctx context.Context, repo RepoContext) (string, error) {
	return p.complete(ctx, analyzePrompt(repo))
}

func (p *geminiProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, reportPrompt(summaries))
}

func (p *geminiProvider) complete(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("Gemini API 호출 실패: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("Gemini API 응답이 비어있습니다")
	}

	return text, nil
}
