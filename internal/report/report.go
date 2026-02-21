package report

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
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

	fmt.Fprintf(g.output, "\n브리핑 리포트 생성 중 (%d개 저장소)...\n\n", len(summaries))

	report, err := g.provider.Report(ctx, summaries)
	if err != nil {
		return fmt.Errorf("리포트 생성 실패: %w", err)
	}

	fmt.Fprintln(g.output, report)
	return nil
}

// WriteToFile은 리포트를 파일에 저장합니다.
func WriteToFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, content)
	return err
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
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	fmt.Fprintln(w, "언어별 통계")
	fmt.Fprintln(w, strings.Repeat("-", 30))
	for _, lc := range sorted {
		bar := strings.Repeat("█", lc.count)
		fmt.Fprintf(w, "%-20s %3d  %s\n", lc.lang, lc.count, bar)
	}
}
