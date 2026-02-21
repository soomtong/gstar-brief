package cmd

import (
	"fmt"
	"os"

	"github.com/dp/gstar-brief/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "gstar-brief",
	Short: "GitHub 스타 저장소 브리핑 CLI",
	Long: `GitHub에서 스타 마킹한 저장소를 분석하고 LLM으로 종합 브리핑 리포트를 생성합니다.

설정 파일: ` + config.DefaultPath() + `
설정 초기화: gstar-brief init`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		fmt.Sprintf("설정 파일 경로 (기본: %s)", config.DefaultPath()))
	rootCmd.PersistentFlags().StringP("output", "o", "", "출력 파일 경로 (기본: stdout)")
	rootCmd.PersistentFlags().IntP("limit", "n", 0, "처리할 저장소 최대 수 (기본: 전체)")
	rootCmd.PersistentFlags().Int("offset", 0, "건너뛸 저장소 수 (기본: 0, 최신 순 기준)")

	// 플래그 → viper 바인딩 (플래그가 최우선)
	viper.BindPFlag("limit", rootCmd.PersistentFlags().Lookup("limit"))
	viper.BindPFlag("offset", rootCmd.PersistentFlags().Lookup("offset"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	config.Init(cfgFile)

	// 설정 파일 경로 출력 (디버그용)
	if cf := viper.ConfigFileUsed(); cf != "" {
		fmt.Fprintf(os.Stderr, "설정 파일: %s\n", cf)
	}

	// 설정값을 SDK가 읽는 환경변수에 적용
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: 설정 로드 실패: %v\n", err)
		return
	}
	cfg.ApplyToEnv()
}
