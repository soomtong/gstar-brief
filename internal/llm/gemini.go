package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

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
	return p.complete(ctx, "analyze", repo.FullName, analyzePrompt(repo))
}

func (p *geminiProvider) Report(ctx context.Context, summaries []RepoSummary) (string, error) {
	return p.complete(ctx, "report", "", reportPrompt(summaries))
}

func (p *geminiProvider) complete(ctx context.Context, action, target, prompt string) (string, error) {
	slog.Debug("LLM API 요청",
		"provider", "gemini",
		"model", p.model,
		"action", action,
		"target", target,
		"prompt_len", len(prompt),
	)

	start := time.Now()
	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(prompt), nil)
	elapsed := time.Since(start)

	if err != nil {
		slog.Debug("LLM API 오류",
			"provider", "gemini",
			"model", p.model,
			"action", action,
			"target", target,
			"duration_ms", elapsed.Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("Gemini API 호출 실패: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("Gemini API 응답이 비어있습니다")
	}

	slog.Debug("LLM API 응답",
		"provider", "gemini",
		"model", p.model,
		"action", action,
		"target", target,
		"duration_ms", elapsed.Milliseconds(),
		"input_tokens", resp.UsageMetadata.PromptTokenCount,
		"output_tokens", resp.UsageMetadata.CandidatesTokenCount,
	)

	return text, nil
}
