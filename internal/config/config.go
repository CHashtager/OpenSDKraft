package config

import (
	"errors"
	"github.com/spf13/viper"
)

type Config struct {
	SDKName       string               `yaml:"sdkName"`
	OutputDir     string               `yaml:"outputDir"`
	PackageName   string               `yaml:"packageName"`
	CodeStyle     CodeStyle            `yaml:"codeStyle"`
	Generator     GeneratorOptions     `yaml:"generator"`
	Testing       Testing              `yaml:"testing"`
	Documentation DocumentationOptions `yaml:"documentation"`
}
type CodeStyle struct {
	UsePointers   bool              `yaml:"usePointers"`
	IndentStyle   string            `yaml:"indentStyle"`
	MaxLineLength int               `yaml:"maxLineLength"`
	Comments      bool              `yaml:"generateComments"`
	Formatting    FormattingOptions `yaml:"formatting"`
}
type Testing struct {
	Generate  bool            `yaml:"generate"`
	Framework string          `yaml:"framework"` // e.g., "testify", "standard"
	Mocks     bool            `yaml:"mocks"`
	Benchmark bool            `yaml:"benchmark"`
	Coverage  CoverageOptions `yaml:"coverage"`
}

type FormattingOptions struct {
	TabWidth       int    `yaml:"tabWidth"`
	UseSpaces      bool   `yaml:"useSpaces"`
	LineEnding     string `yaml:"lineEnding"`
	RemoveComments bool   `yaml:"removeComments"`
}

type ClientOptions struct {
	Timeout             int  `yaml:"timeout"`
	RetryEnabled        bool `yaml:"retryEnabled"`
	MaxRetries          int  `yaml:"maxRetries"`
	RetryBackoffSeconds int  `yaml:"retryBackoffSeconds"`
	UseContext          bool `yaml:"useContext"`
	GenerateMiddleware  bool `yaml:"generateMiddleware"`
	IncludeRateLimiting bool `yaml:"includeRateLimiting"`
}

type GeneratorOptions struct {
	IncludeExamples    bool          `yaml:"includeExamples"`
	IncludeValidation  bool          `yaml:"includeValidation"`
	GenerateInterfaces bool          `yaml:"generateInterfaces"`
	Verbose            bool          `yaml:"verbose"`
	ClientOptions      ClientOptions `yaml:"clientOptions"`
}

type TestingOptions struct {
	Generate          bool            `yaml:"generate"`
	Framework         string          `yaml:"framework"`
	GenerateMocks     bool            `yaml:"generateMocks"`
	IncludeBenchmarks bool            `yaml:"includeBenchmarks"`
	Coverage          CoverageOptions `yaml:"coverage"`
}

type CoverageOptions struct {
	Enabled     bool     `yaml:"enabled"`
	Threshold   float64  `yaml:"threshold"`
	ExcludeList []string `yaml:"excludeList"`
}

type DocumentationOptions struct {
	Generate         bool   `yaml:"generate"`
	Format           string `yaml:"format"`
	IncludeExamples  bool   `yaml:"includeExamples"`
	OutputFormat     string `yaml:"outputFormat"`
	IncludeChangelog bool   `yaml:"includeChangelog"`
}

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("sdkName", "generatedSDK")
	v.SetDefault("outputDir", "./generated")
	v.SetDefault("codeStyle.usePointers", true)
	v.SetDefault("codeStyle.indentStyle", "space")
	v.SetDefault("codeStyle.maxLineLength", 120)
	v.SetDefault("codeStyle.generateComments", true)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	var config Config
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
		// Use defaults if no config file found
	}

	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
