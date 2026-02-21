package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dp/gstar-brief/internal/github"
	"github.com/dp/gstar-brief/internal/llm"
)

const workerCount = 3 // GitHub Rate Limit 고려한 동시 요청 수

// Analyzer는 GitHub 저장소를 수집하고 LLM으로 분석합니다.
type Analyzer struct {
	github   *github.Client
	provider llm.Provider
}

// New는 새 Analyzer를 생성합니다.
func New(gh *github.Client, provider llm.Provider) *Analyzer {
	return &Analyzer{
		github:   gh,
		provider: provider,
	}
}

// Run은 username의 스타 저장소를 수집하고 분석하여 요약 목록을 반환합니다.
func (a *Analyzer) Run(ctx context.Context, username string, limit int) ([]llm.RepoSummary, error) {
	slog.Info("GitHub 스타 저장소 수집 중", "user", username)

	repos, err := a.github.ListStarred(ctx, username, limit)
	if err != nil {
		return nil, fmt.Errorf("스타 저장소 수집 실패: %w", err)
	}

	slog.Info("저장소 분석 시작", "count", len(repos))

	type result struct {
		idx     int
		summary llm.RepoSummary
		err     error
	}

	jobs := make(chan int, len(repos))
	results := make(chan result, len(repos))

	// 워커 풀
	var wg sync.WaitGroup
	for range workerCount {
		wg.Go(func() {
			for idx := range jobs {
				repo := repos[idx]

				readme, _ := a.github.GetReadme(ctx, repo.FullName)
				snippets, _ := a.github.GetCodeSnippets(ctx, repo.FullName)

				repoCtx := llm.RepoContext{
					Name:         repo.Name,
					FullName:     repo.FullName,
					Description:  repo.Description,
					Language:     repo.Language,
					Stars:        repo.StargazersCount,
					StarredAt:    repo.StarredAt,
					Topics:       repo.Topics,
					Readme:       readme,
					CodeSnippets: snippets,
				}

				summary, err := a.provider.Analyze(ctx, repoCtx)
				if err != nil {
					results <- result{idx: idx, err: fmt.Errorf("%s 분석 실패: %w", repo.FullName, err)}
					return
				}

				slog.Info("저장소 분석 완료", "repo", repo.FullName, "idx", idx+1, "total", len(repos))

				results <- result{
					idx: idx,
					summary: llm.RepoSummary{
						FullName:  repo.FullName,
						Language:  repo.Language,
						Stars:     repo.StargazersCount,
						StarredAt: repo.StarredAt,
						Summary:   summary,
					},
				}
			}
		})
	}

	// 작업 분배
	for i := range len(repos) {
		jobs <- i
	}
	close(jobs)

	// 완료 대기 후 채널 닫기
	go func() {
		wg.Wait()
		close(results)
	}()

	// 결과 수집 (원래 순서 유지)
	summaries := make([]llm.RepoSummary, len(repos))
	for r := range results {
		if r.err != nil {
			slog.Warn("저장소 분석 실패", "error", r.err)
			continue
		}
		summaries[r.idx] = r.summary
	}

	// 빈 요약 제거 (오류로 인해 채워지지 않은 슬롯)
	var filtered []llm.RepoSummary
	for _, s := range summaries {
		if s.FullName != "" {
			filtered = append(filtered, s)
		}
	}

	return filtered, nil
}
