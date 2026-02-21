package report

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/dp/gstar-brief/internal/llm"
)

// Generator는 브리핑 리포트를 생성합니다.
type Generator struct {
	provider llm.Provider
	output   io.Writer
}

// New는 새 Generator를 생성합니다.
// output이 nil이면 os.Stdout을 사용합니다.
func New(provider llm.Provider, output io.Writer) *Generator {
	if output == nil {
		output = os.Stdout
	}
	return &Generator{
		provider: provider,
		output:   output,
	}
}

// isTerminal은 w가 터미널(os.Stdout)인지 확인합니다.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Generate는 요약 목록을 LLM에 전달하여 종합 브리핑 리포트를 생성하고 출력합니다.
// 출력 대상이 터미널이면 glamour로 마크다운을 렌더링합니다.
func (g *Generator) Generate(ctx context.Context, summaries []llm.RepoSummary) error {
	if len(summaries) == 0 {
		return fmt.Errorf("분석된 저장소가 없습니다")
	}

	slog.Info("브리핑 리포트 생성 중", "count", len(summaries))
	fmt.Fprintln(g.output)

	content, err := g.provider.Report(ctx, summaries)
	if err != nil {
		return fmt.Errorf("리포트 생성 실패: %w", err)
	}

	if isTerminal(g.output) {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)
		if err == nil {
			rendered, rerr := renderer.Render(content)
			if rerr == nil {
				fmt.Fprint(g.output, rendered)
				return nil
			}
		}
		// 렌더링 실패 시 raw 출력으로 폴백
		slog.Warn("glamour 렌더링 실패, raw 출력으로 대체", "err", err)
	}

	fmt.Fprintln(g.output, content)
	return nil
}

// WriteToFile은 리포트를 파일에 저장합니다.
func WriteToFile(path string, content string) error {
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}
	return nil
}

// LangStats는 언어별 저장소 수 통계를 반환합니다.
func LangStats(summaries []llm.RepoSummary) map[string]int {
	stats := make(map[string]int)
	for _, s := range summaries {
		lang := "미분류"
		if s.Language != nil && *s.Language != "" {
			lang = *s.Language
		}
		stats[lang]++
	}
	return stats
}

// PrintLangStats는 언어별 통계를 테이블 형식으로 출력합니다.
func PrintLangStats(w io.Writer, summaries []llm.RepoSummary) {
	stats := LangStats(summaries)

	type langCount struct {
		lang  string
		count int
	}

	var sorted []langCount
	for lang, count := range stats {
		sorted = append(sorted, langCount{lang, count})
	}
	slices.SortFunc(sorted, func(a, b langCount) int {
		return b.count - a.count
	})

	fmt.Fprintln(w, "언어별 통계")
	fmt.Fprintln(w, strings.Repeat("-", 30))
	for _, lc := range sorted {
		bar := strings.Repeat("█", lc.count)
		fmt.Fprintf(w, "%-20s %3d  %s\n", lc.lang, lc.count, bar)
	}
}
