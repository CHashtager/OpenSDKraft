package main

import (
	"fmt"
	"log"
	"os"

	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/generator"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	cfgFile string
	verbose bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "sdkraft [flags] <openapi-file>",
		Short:         "Generate Go SDK from OpenAPI specification",
		Version:       version,
		Args:          cobra.ExactArgs(1),
		RunE:          runGenerate,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringP("output", "o", "", "output directory")
	rootCmd.PersistentFlags().StringP("package", "p", "", "package name for generated code")
	rootCmd.PersistentFlags().Bool("with-tests", true, "generate tests")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set verbose mode from flag
	cfg.Generator.Verbose = verbose

	err = cfg.Validate()
	if err != nil {
		return err
	}

	// Override config with command line flags
	if outputDir, _ := cmd.Flags().GetString("output"); outputDir != "" {
		cfg.OutputDir = outputDir
	}
	if packageName, _ := cmd.Flags().GetString("package"); packageName != "" {
		cfg.PackageName = packageName
	}
	if withTests, _ := cmd.Flags().GetBool("with-tests"); !withTests {
		cfg.Testing.Generate = false
	}

	// Initialize generator
	gen, err := generator.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	defer gen.Close()

	// Generate SDK
	if err := gen.Generate(args[0]); err != nil {
		return fmt.Errorf("failed to generate SDK: %w", err)
	}

	log.Printf("Successfully generated SDK in %s", cfg.OutputDir)
	return nil
}
