package report

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

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

// Generate는 요약 목록을 LLM에 전달하여 종합 브리핑 리포트를 생성하고 출력합니다.
func (g *Generator) Generate(ctx context.Context, summaries []llm.RepoSummary) error {
	if len(summaries) == 0 {
		return fmt.Errorf("분석된 저장소가 없습니다")
	}

	slog.Info("브리핑 리포트 생성 중", "count", len(summaries))
	fmt.Fprintln(g.output)

	report, err := g.provider.Report(ctx, summaries)
	if err != nil {
		return fmt.Errorf("리포트 생성 실패: %w", err)
	}

	fmt.Fprintln(g.output, report)
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
