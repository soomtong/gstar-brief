package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dp/gstar-brief/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "설정 파일 초기화",
	Long: fmt.Sprintf(`기본 설정 파일을 생성합니다.

생성 위치: %s

이미 파일이 존재하면 덮어쓰지 않습니다. --force 플래그로 강제 덮어쓰기 가능합니다.`, config.DefaultPath()),
	RunE: runInit,
}

func init() {
	initCmd.Flags().Bool("force", false, "기존 설정 파일을 덮어씁니다")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")
	path := config.DefaultPath()

	// 이미 존재하는 경우
	if _, err := os.Stat(path); err == nil && !force {
		fmt.Fprintf(cmd.OutOrStdout(), "설정 파일이 이미 존재합니다: %s\n", path)
		fmt.Fprintln(cmd.OutOrStdout(), "덮어쓰려면 --force 플래그를 사용하세요.")
		return nil
	}

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("설정 디렉토리 생성 실패: %w", err)
	}

	// 파일 작성
	if err := os.WriteFile(path, []byte(config.DefaultTOML()), 0o600); err != nil {
		return fmt.Errorf("설정 파일 작성 실패: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "설정 파일 생성 완료: %s\n\n", path)
	fmt.Fprintln(cmd.OutOrStdout(), "다음 항목을 편집하여 설정을 완료하세요:")
	fmt.Fprintln(cmd.OutOrStdout(), "  [github] token, user")
	fmt.Fprintln(cmd.OutOrStdout(), "  [llm]    provider, 해당 API 키")
	return nil
}
