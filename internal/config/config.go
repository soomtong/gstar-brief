// Package config는 gstar-brief의 설정을 관리합니다.
//
// 우선순위 (높음 → 낮음):
//  1. CLI 플래그 (--limit, --output 등)
//  2. 환경변수 GSTAR_* (예: GSTAR_GITHUB_TOKEN)
//  3. 레거시 환경변수 (GITHUB_TOKEN, LLM_PROVIDER 등)
//  4. 설정 파일 (~/.config/gstar-brief/config.toml)
//  5. 기본값
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	AppName   = "gstar-brief"
	EnvPrefix = "GSTAR"
)

// Config는 gstar-brief의 전체 설정입니다.
type Config struct {
	GitHub GitHubConfig `mapstructure:"github"`
	LLM    LLMConfig    `mapstructure:"llm"`
}

// GitHubConfig는 GitHub API 관련 설정입니다.
type GitHubConfig struct {
	Token string `mapstructure:"token"`
	User  string `mapstructure:"user"`
}

// LLMConfig는 LLM provider 관련 설정입니다.
type LLMConfig struct {
	Provider        string `mapstructure:"provider"`
	Model           string `mapstructure:"model"`
	AnthropicKey    string `mapstructure:"anthropic_key"`
	OpenAIKey       string `mapstructure:"openai_key"`
	GeminiKey       string `mapstructure:"gemini_key"`
	OpenRouterKey   string `mapstructure:"openrouter_key"`
	OpenRouterModel string `mapstructure:"openrouter_model"`
	OllamaBaseURL   string `mapstructure:"ollama_base_url"`
	OllamaModel     string `mapstructure:"ollama_model"`
}

// Dir는 설정 디렉토리 경로를 반환합니다.
// 우선순위: GSTAR_CONFIG_DIR > XDG_CONFIG_HOME/gstar-brief > ~/.config/gstar-brief
func Dir() string {
	if dir := os.Getenv("GSTAR_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", AppName)
	}
	return filepath.Join(home, ".config", AppName)
}

// DefaultPath는 기본 설정 파일 경로를 반환합니다.
func DefaultPath() string {
	return filepath.Join(Dir(), "config.toml")
}

// Init은 viper를 초기화합니다. cfgFile이 비어있으면 기본 경로를 탐색합니다.
func Init(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// XDG / ~/.config 탐색
		viper.AddConfigPath(Dir())
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	// GSTAR_ 접두사 환경변수 자동 매핑
	// 예: GSTAR_GITHUB_TOKEN → github.token
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 레거시 환경변수 폴백 바인딩
	bindLegacyEnv()

	// 기본값
	setDefaults()

	// 설정 파일 읽기 (없어도 계속 진행)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "경고: 설정 파일 파싱 오류: %v\n", err)
		}
	}
}

// bindLegacyEnv는 기존 환경변수를 viper 키에 폴백으로 바인딩합니다.
// GSTAR_* 가 설정되지 않은 경우에만 레거시 환경변수가 사용됩니다.
func bindLegacyEnv() {
	legacyBindings := map[string]string{
		"github.token":         "GITHUB_TOKEN",
		"github.user":          "GITHUB_USER",
		"llm.provider":         "LLM_PROVIDER",
		"llm.model":            "LLM_MODEL",
		"llm.anthropic_key":    "ANTHROPIC_API_KEY",
		"llm.openai_key":       "OPENAI_API_KEY",
		"llm.gemini_key":       "GEMINI_API_KEY",
		"llm.openrouter_key":   "OPENROUTER_API_KEY",
		"llm.openrouter_model": "OPENROUTER_MODEL",
		"llm.ollama_base_url":  "OLLAMA_BASE_URL",
		"llm.ollama_model":     "OLLAMA_MODEL",
	}

	for key, envVar := range legacyBindings {
		// GSTAR_* 가 없을 때만 레거시 값 사용
		gstarKey := EnvPrefix + "_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if os.Getenv(gstarKey) == "" {
			if val := os.Getenv(envVar); val != "" {
				viper.SetDefault(key, val)
			}
		}
	}
}

func setDefaults() {
	viper.SetDefault("llm.ollama_base_url", "http://localhost:11434")
	viper.SetDefault("llm.openrouter_model", "openai/gpt-5-nano")
}

// Load는 현재 viper 설정을 Config 구조체로 반환합니다.
func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("설정 파싱 실패: %w", err)
	}
	return &cfg, nil
}

// ApplyToEnv는 설정값을 SDK가 읽는 환경변수로 내보냅니다.
// anthropic-sdk-go, openai-go, genai 등은 표준 환경변수를 직접 읽으므로 필요합니다.
func (c *Config) ApplyToEnv() {
	setIfNotEmpty := func(key, val string) {
		if val != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}

	setIfNotEmpty("GITHUB_TOKEN", c.GitHub.Token)
	setIfNotEmpty("GITHUB_USER", c.GitHub.User)
	setIfNotEmpty("LLM_PROVIDER", c.LLM.Provider)
	setIfNotEmpty("LLM_MODEL", c.LLM.Model)
	setIfNotEmpty("ANTHROPIC_API_KEY", c.LLM.AnthropicKey)
	setIfNotEmpty("OPENAI_API_KEY", c.LLM.OpenAIKey)
	setIfNotEmpty("GEMINI_API_KEY", c.LLM.GeminiKey)
	setIfNotEmpty("OPENROUTER_API_KEY", c.LLM.OpenRouterKey)
	setIfNotEmpty("OPENROUTER_MODEL", c.LLM.OpenRouterModel)
	setIfNotEmpty("OLLAMA_BASE_URL", c.LLM.OllamaBaseURL)
	setIfNotEmpty("OLLAMA_MODEL", c.LLM.OllamaModel)
}

// DefaultTOML은 초기 설정 파일의 TOML 내용을 반환합니다.
func DefaultTOML() string {
	return `# gstar-brief 설정 파일
# 우선순위: CLI 플래그 > GSTAR_* 환경변수 > 이 파일 > 기본값

[github]
# GitHub Personal Access Token (public_repo 스코프 필요)
# 발급: https://github.com/settings/tokens
token = ""

# 분석할 GitHub 유저명
user = ""

[llm]
# 사용할 LLM provider: claude | openai | gemini | openrouter | ollama
provider = "gemini"

# 모델 오버라이드 (비워두면 provider 기본값 사용)
# claude   기본값: claude-haiku-4-5
# openai   기본값: gpt-5-nano
# gemini   기본값: gemini-2.5-flash-lite
# openrouter 기본값: openai/gpt-5-nano
model = ""

# Anthropic API Key (provider = "claude" 시 필요)
anthropic_key = ""

# OpenAI API Key (provider = "openai" 시 필요)
openai_key = ""

# Google Gemini API Key (provider = "gemini" 시 필요)
gemini_key = ""

# OpenRouter API Key (provider = "openrouter" 시 필요)
openrouter_key = ""

# OpenRouter 모델 ID (기본: openai/gpt-5-nano)
openrouter_model = ""

# Ollama 엔드포인트 (기본: http://localhost:11434)
ollama_base_url = ""

# Ollama 모델명 (provider = "ollama" 시 필요)
ollama_model = ""
`
}
