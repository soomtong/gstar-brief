package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dp/gstar-brief/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var verbose bool

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
	rootCmd.PersistentFlags().IntP("limit", "n", 100, "처리할 저장소 최대 수 (기본: 100, 전체: 0)")
	rootCmd.PersistentFlags().Int("offset", 0, "건너뛸 저장소 수 (기본: 0, 최신 순 기준)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "LLM API 호출 상세 로그 출력")

	// 플래그 → viper 바인딩 (플래그가 최우선)
	viper.BindPFlag("limit", rootCmd.PersistentFlags().Lookup("limit"))
	viper.BindPFlag("offset", rootCmd.PersistentFlags().Lookup("offset"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	// verbose 플래그에 따라 slog 레벨 설정
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	config.Init(cfgFile)

	// 설정 파일 경로 출력 (디버그용)
	if cf := viper.ConfigFileUsed(); cf != "" {
		fmt.Fprintf(os.Stderr, "설정 파일: %s\n\n", cf)
	}

	// 설정값을 SDK가 읽는 환경변수에 적용
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: 설정 로드 실패: %v\n", err)
		return
	}
	cfg.ApplyToEnv()
}
