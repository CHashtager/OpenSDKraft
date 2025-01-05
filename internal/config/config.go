package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

type Config struct {
	SDKName       string               `yaml:"sdkName"`
	OutputDir     string               `yaml:"outputDir"`
	PackageName   string               `yaml:"packageName"`
	Module        string               `yaml:"module"`
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
	UseAuth             bool `yaml:"useAuth"`
}

type GeneratorOptions struct {
	IncludeExamples    bool          `yaml:"includeExamples"`
	IncludeValidation  bool          `yaml:"includeValidation"`
	GenerateInterfaces bool          `yaml:"generateInterfaces"`
	IncludeJSON        bool          `yaml:"includeJSON"`
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

func (c *Config) Validate() error {
	if err := c.validatePaths(); err != nil {
		return err
	}

	if err := c.validateGenerator(); err != nil {
		return err
	}

	if err := c.validateTesting(); err != nil {
		return err
	}

	return c.validateCodeStyle()
}

func (c *Config) validatePaths() error {
	if c.OutputDir == "" {
		c.OutputDir = "./generated"
	}

	// Expand environment variables
	c.OutputDir = os.ExpandEnv(c.OutputDir)

	return nil
}

func (c *Config) validateGenerator() error {
	if c.Generator.ClientOptions.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if c.Generator.ClientOptions.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}

func (c *Config) validateTesting() error {
	if c.Testing.Generate {
		if c.Testing.Framework == "" {
			c.Testing.Framework = "testify"
		}

		if c.Testing.Coverage.Threshold < 0 || c.Testing.Coverage.Threshold > 100 {
			return fmt.Errorf("coverage threshold must be between 0 and 100")
		}
	}

	return nil
}

func (c *Config) validateCodeStyle() error {
	if c.CodeStyle.MaxLineLength < 0 {
		return fmt.Errorf("max line length cannot be negative")
	}

	if c.CodeStyle.IndentStyle != "tab" && c.CodeStyle.IndentStyle != "space" {
		return fmt.Errorf("indent style must be either 'tab' or 'space'")
	}

	return nil
}

func (c *Config) ApplyDefaults() {
	if c.SDKName == "" {
		c.SDKName = filepath.Base(c.OutputDir)
	}

	if c.PackageName == "" {
		c.PackageName = "sdk"
	}

	if c.Generator.ClientOptions.Timeout == 0 {
		c.Generator.ClientOptions.Timeout = 30
	}

	if c.CodeStyle.MaxLineLength == 0 {
		c.CodeStyle.MaxLineLength = 120
	}
}
