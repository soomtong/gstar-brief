package cmd

import (
	"context"
	"fmt"
	"slices"

	"github.com/dp/gstar-brief/internal/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "스타 저장소 목록 출력",
	Long:  "GitHub 스타 저장소 목록을 정렬하여 출력합니다.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringP("sort", "s", "date", "정렬 기준: stars | date")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	username := viper.GetString("github.user")
	if username == "" {
		return fmt.Errorf("GitHub 유저명이 설정되지 않았습니다\n설정 방법:\n  환경변수: GSTAR_GITHUB_USER=username\n  설정 파일: gstar-brief init 후 [github] user 항목 편집")
	}

	sortBy, _ := cmd.Flags().GetString("sort")
	limit := viper.GetInt("limit")

	client := github.New(viper.GetString("github.token"))
	ctx := context.Background()

	repos, err := client.ListStarred(ctx, username, limit)
	if err != nil {
		return err
	}

	switch sortBy {
	case "stars":
		slices.SortFunc(repos, func(a, b github.Repo) int {
			return b.StargazersCount - a.StargazersCount
		})
	case "date":
		slices.SortFunc(repos, func(a, b github.Repo) int {
			return b.StarredAt.Compare(a.StarredAt)
		})
	default:
		return fmt.Errorf("알 수 없는 정렬 기준: %q (stars | date)", sortBy)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%-50s  %7s  %-14s  %s\n", "저장소", "스타", "언어", "스타 날짜")
	fmt.Fprintf(w, "%s\n", "──────────────────────────────────────────────────────────────────────────────────────────────")

	for _, r := range repos {
		lang := "-"
		if r.Language != nil {
			lang = *r.Language
		}
		fmt.Fprintf(w, "%-50s  %7d  %-14s  %s\n",
			r.FullName,
			r.StargazersCount,
			lang,
			r.StarredAt.Format("2006-01-02"),
		)
	}

	fmt.Fprintf(w, "\n총 %d개\n", len(repos))
	return nil
}
