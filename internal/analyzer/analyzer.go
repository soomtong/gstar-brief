package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/dp/gstar-brief/internal/github"
	"github.com/dp/gstar-brief/internal/llm"
)

const workerCount = 3 // GitHub Rate Limit 고려한 동시 요청 수

// statusPrinter는 glamour 렌더러와 출력 mutex를 보유합니다.
type statusPrinter struct {
	mu       sync.Mutex
	renderer *glamour.TermRenderer
}

// newStatusPrinter는 터미널 쿼리 없이 고정 스타일로 렌더러를 생성합니다.
func newStatusPrinter() *statusPrinter {
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return &statusPrinter{}
	}
	return &statusPrinter{renderer: r}
}

// print는 마크다운을 렌더링하여 stderr에 출력합니다 (goroutine-safe).
func (p *statusPrinter) print(md string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.renderer == nil {
		fmt.Fprintln(os.Stderr, md)
		return
	}
	out, err := p.renderer.Render(md)
	if err != nil {
		fmt.Fprintln(os.Stderr, md)
		return
	}
	fmt.Fprint(os.Stderr, out)
}

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
// offset: 건너뛸 아이템 수 (최신 순 기준)
// limit:  수집할 최대 아이템 수 (0이면 무제한)
func (a *Analyzer) Run(ctx context.Context, username string, offset, limit int) ([]llm.RepoSummary, error) {
	sp := newStatusPrinter()

	slog.Debug("GitHub 스타 저장소 수집 중", "user", username)
	sp.print(fmt.Sprintf("**@%s** 의 스타 저장소를 수집하는 중...", username))

	repos, err := a.github.ListStarred(ctx, username, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("스타 저장소 수집 실패: %w", err)
	}

	slog.Debug("저장소 분석 시작", "count", len(repos))
	sp.print(fmt.Sprintf("`%d`개 저장소 분석을 시작합니다.", len(repos)))

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

				slog.Debug("저장소 분석 완료", "repo", repo.FullName, "idx", idx+1, "total", len(repos))
				sp.print(fmt.Sprintf("**[%d/%d]** `%s` 분석 완료", idx+1, len(repos), repo.FullName))

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
			sp.print(fmt.Sprintf("> **경고:** %s", r.err))
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
