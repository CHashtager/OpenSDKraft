package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/logging"
	"github.com/getkin/kin-openapi/openapi3"
	"os"
	"path/filepath"

	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/parser"
)

type Generator struct {
	config         *config.Config
	parser         *parser.Parser
	modelGen       *ModelGenerator
	operationGen   *OperationGenerator
	templateEngine *TemplateEngine
	validator      *Validator
	codeValidator  *CodeValidator
	logger         *logging.Logger
}

// New creates a new Generator instance with all required components
func New(cfg *config.Config) (*Generator, error) {
	// Initialize logger
	logger, err := logging.NewLogger(
		filepath.Join(cfg.OutputDir, "generation.log"),
		logging.INFO,
		cfg.Generator.Verbose,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize parser
	p, err := parser.New()
	if err != nil {
		logger.Error("Failed to initialize parser: %v", err)
		return nil, fmt.Errorf("failed to initialize parser: %w", err)
	}

	// Initialize template engine
	tmplEngine, err := NewTemplateEngine(cfg)
	if err != nil {
		logger.Error("Failed to initialize template engine: %v", err)
		return nil, fmt.Errorf("failed to initialize template engine: %w", err)
	}

	// Create generator instance
	g := &Generator{
		config:         cfg,
		parser:         p,
		templateEngine: tmplEngine,
		validator:      NewValidator(),
		codeValidator:  NewCodeValidator(cfg),
		logger:         logger,
	}

	// Initialize model and operation generators
	g.modelGen = NewModelGenerator(cfg, tmplEngine, logger)
	g.operationGen = NewOperationGenerator(cfg, tmplEngine, logger)

	logger.Info("Generator initialized successfully")
	return g, nil
}

// writeAndValidateFile writes content to a file and validates it if it's a Go file
func (g *Generator) writeAndValidateFile(filename string, content []byte) error {
	// Always validate Go files before writing
	if filepath.Ext(filename) == ".go" {
		g.logger.Debug("Validating generated code for: %s", filename)
		if err := g.codeValidator.ValidateGoCode(filename, content); err != nil {
			g.logger.Error("Code validation failed for %s: %v", filename, err)
			return fmt.Errorf("code validation failed for %s: %w", filename, err)
		}
		g.logger.Debug("Code validation successful for: %s", filename)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the file
	if err := os.WriteFile(filename, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// Generate processes the OpenAPI specification and generates the SDK
func (g *Generator) Generate(inputFile string) error {
	g.logger.Info("Starting SDK generation from file: %s", inputFile)

	// Load and parse OpenAPI document
	g.logger.Debug("Parsing OpenAPI document")
	doc, err := g.parser.ParseFile(inputFile)
	if err != nil {
		g.logger.Error("Failed to parse OpenAPI document: %v", err)
		return fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}
	g.logger.Info("Successfully parsed OpenAPI document")

	// Validate the document
	g.logger.Debug("Validating OpenAPI document")
	if err := g.validator.ValidateDocument(doc); err != nil {
		g.logger.Error("Document validation failed: %v", err)
		return fmt.Errorf("document validation failed: %w", err)
	}
	g.logger.Info("OpenAPI document validation successful")

	// Create output directory structure
	g.logger.Debug("Creating output directory structure")
	if err := g.createOutputStructure(); err != nil {
		g.logger.Error("Failed to create output structure: %v", err)
		return fmt.Errorf("failed to create output directory structure: %w", err)
	}

	// Set up validation results collection
	validationErrors := make([]error, 0)

	// Generate and validate models
	g.logger.Info("Generating and validating models")
	if err := g.generateAndValidateModels(doc.Components.Schemas); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Generate and validate operations
	g.logger.Info("Generating and validating operations")
	if err := g.generateAndValidateOperations(*doc.Paths); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Generate and validate tests if enabled
	if g.config.Testing.Generate {
		g.logger.Info("Generating and validating tests")
		if err := g.generateAndValidateTests(g.operationGen.GetOperations()); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	// Report validation errors
	if len(validationErrors) > 0 {
		g.logger.Error("Generation completed with validation errors:")
		for _, err := range validationErrors {
			g.logger.Error("- %v", err)
		}
		return fmt.Errorf("generation completed with %d validation errors", len(validationErrors))
	}

	g.logger.Info("SDK generation completed successfully")
	return nil
}

func (g *Generator) generateAndValidateModels(schemas openapi3.Schemas) error {
	modelsDir := filepath.Join(g.config.OutputDir, "models")
	if err := g.modelGen.Generate(schemas); err != nil {
		return fmt.Errorf("failed to generate models: %w", err)
	}

	// Validate all generated model files
	return g.validateGeneratedFiles(modelsDir)
}

func (g *Generator) generateAndValidateOperations(paths openapi3.Paths) error {
	operationsDir := filepath.Join(g.config.OutputDir, "operations")
	if err := g.operationGen.Generate(&paths); err != nil {
		return fmt.Errorf("failed to generate operations: %w", err)
	}

	// Validate all generated operation files
	return g.validateGeneratedFiles(operationsDir)
}

func (g *Generator) generateAndValidateTests(operations []*Operation) error {
	testsDir := filepath.Join(g.config.OutputDir, "tests")
	testGen := NewTestGenerator(g.config, g.templateEngine, g.parser)
	if err := testGen.Generate(operations); err != nil {
		return fmt.Errorf("failed to generate tests: %w", err)
	}

	// Validate all generated test files
	return g.validateGeneratedFiles(testsDir)
}

func (g *Generator) validateGeneratedFiles(dir string) error {
	var validationErrors []error

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".go" {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			if err := g.codeValidator.ValidateGoCode(path, content); err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("validation failed for %s: %w", path, err))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation failed with %d errors", len(validationErrors))
	}

	return nil
}

// createOutputStructure creates the necessary directory structure for generated code
func (g *Generator) createOutputStructure() error {
	dirs := []string{
		g.config.OutputDir,
		filepath.Join(g.config.OutputDir, "models"),
		filepath.Join(g.config.OutputDir, "operations"),
	}

	if g.config.Testing.Generate {
		dirs = append(dirs, filepath.Join(g.config.OutputDir, "tests"))
	}

	for _, dir := range dirs {
		g.logger.Debug("Creating directory: %s", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Close cleans up resources used by the generator
func (g *Generator) Close() error {
	if g.logger != nil {
		return g.logger.Close()
	}
	return nil
}
