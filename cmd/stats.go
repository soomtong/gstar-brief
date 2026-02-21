package cmd

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/dp/gstar-brief/internal/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "언어별 통계 출력",
	Long:  "스타 저장소를 언어별로 집계하여 통계를 출력합니다.",
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, _ []string) error {
	username := viper.GetString("github.user")
	if username == "" {
		return fmt.Errorf("GitHub 유저명이 설정되지 않았습니다\n설정 방법:\n  환경변수: GSTAR_GITHUB_USER=username\n  설정 파일: gstar-brief init 후 [github] user 항목 편집")
	}

	limit := viper.GetInt("limit")
	client := github.New(viper.GetString("github.token"))
	ctx := context.Background()

	repos, err := client.ListStarred(ctx, username, limit)
	if err != nil {
		return err
	}

	// 언어별 집계
	langCount := make(map[string]int)
	langStars := make(map[string]int)
	for _, r := range repos {
		lang := "미분류"
		if r.Language != nil && *r.Language != "" {
			lang = *r.Language
		}
		langCount[lang]++
		langStars[lang] += r.StargazersCount
	}

	type stat struct {
		lang  string
		count int
		stars int
	}

	// maps.Keys로 키 목록을 추출한 뒤 slices로 정렬
	langs := slices.Collect(maps.Keys(langCount))
	stats := make([]stat, 0, len(langs))
	for _, lang := range langs {
		stats = append(stats, stat{lang, langCount[lang], langStars[lang]})
	}
	slices.SortFunc(stats, func(a, b stat) int {
		return b.count - a.count
	})

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "@%s 스타 저장소 언어별 통계 (총 %d개)\n\n", username, len(repos))
	fmt.Fprintf(w, "%-20s  %5s  %10s\n", "언어", "저장소", "총 스타")
	fmt.Fprintf(w, "%s\n", strings.Repeat("─", 42))

	maxCount := 0
	if len(stats) > 0 {
		maxCount = stats[0].count
	}

	for _, s := range stats {
		barLen := 0
		if maxCount > 0 {
			barLen = s.count * 20 / maxCount
		}
		bar := strings.Repeat("█", max(barLen, 1))
		fmt.Fprintf(w, "%-20s  %5d  %10d  %s\n", s.lang, s.count, s.stars, bar)
	}

	return nil
}
