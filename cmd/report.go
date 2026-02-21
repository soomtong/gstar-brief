package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dp/gstar-brief/internal/analyzer"
	"github.com/dp/gstar-brief/internal/github"
	"github.com/dp/gstar-brief/internal/llm"
	"github.com/dp/gstar-brief/internal/report"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "LLM 브리핑 리포트 생성",
	Long:  "스타 저장소를 LLM으로 분석하여 종합 브리핑 리포트를 생성합니다.",
	RunE:  runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, _ []string) error {
	username := viper.GetString("github.user")
	if username == "" {
		return fmt.Errorf("GitHub 유저명이 설정되지 않았습니다\n설정 방법:\n  환경변수: GSTAR_GITHUB_USER=username\n  설정 파일: gstar-brief init 후 [github] user 항목 편집")
	}

	offset := viper.GetInt("offset")
	limit := viper.GetInt("limit")
	outputPath := viper.GetString("output")

	// LLM Provider 초기화
	provider, err := llm.New()
	if err != nil {
		return fmt.Errorf("LLM Provider 초기화 실패: %w\n\nLLM 설정 방법:\n  환경변수: GSTAR_LLM_PROVIDER=gemini GSTAR_GEMINI_KEY=AIza...\n  설정 파일: gstar-brief init 후 [llm] 항목 편집", err)
	}

	// GitHub 클라이언트
	ghClient := github.New(viper.GetString("github.token"))
	ctx := context.Background()

	// 저장소 분석
	a := analyzer.New(ghClient, provider)
	summaries, err := a.Run(ctx, username, offset, limit)
	if err != nil {
		return err
	}

	// 출력 대상 결정
	var out *os.File
	if outputPath != "" {
		out, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("출력 파일 생성 실패: %w", err)
		}
		defer out.Close()
		fmt.Fprintf(cmd.OutOrStdout(), "리포트를 %s 에 저장합니다...\n", outputPath)
	} else {
		out = os.Stdout
	}

	// 리포트 생성
	gen := report.New(provider, out)
	return gen.Generate(ctx, summaries)
}
