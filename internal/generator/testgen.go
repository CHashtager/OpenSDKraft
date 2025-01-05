package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/parser"
	"github.com/chashtager/opensdkraft/internal/utils"
	"path/filepath"
	"strings"
)

type TestGenerator struct {
	config    *config.Config
	templates *TemplateEngine
	parser    *parser.Parser
}

type OperationTestData struct {
	Operation   *Operation
	PackageName string
	Config      *config.Config
}

func NewTestGenerator(config *config.Config, templates *TemplateEngine, parser *parser.Parser) *TestGenerator {
	return &TestGenerator{
		config:    config,
		templates: templates,
		parser:    parser,
	}
}

func (g *TestGenerator) Generate(operations []*Operation) error {
	for _, op := range operations {
		if err := g.generateOperationTest(op, g.config.OutputDir); err != nil {
			return fmt.Errorf("failed to generate tests for operation %s: %w", op.Name, err)
		}
	}

	// Generate test helpers
	if err := g.generateTestHelpers(); err != nil {
		return fmt.Errorf("failed to generate test helpers: %w", err)
	}

	return nil
}

func (g *TestGenerator) generateOperationTest(operation *Operation, outputDir string) error {
	data := &OperationTestData{
		Operation:   operation,
		PackageName: g.config.PackageName,
		Config:      g.config,
	}

	content, err := g.templates.Execute("operation_test", data)
	if err != nil {
		return fmt.Errorf("failed to generate test for operation %s: %w", operation.Name, err)
	}

	filename := filepath.Join(outputDir, "tests", strings.ToLower(operation.Name)+"_test.go")
	if err := utils.WriteFile(filename, content); err != nil {
		return fmt.Errorf("failed to write test file for operation %s: %w", operation.Name, err)
	}

	return nil
}

func (g *TestGenerator) generateTestHelpers() error {
	data := struct {
		PackageName string
		Config      *config.Config
	}{
		PackageName: g.config.PackageName,
		Config:      g.config,
	}

	content, err := g.templates.Execute("test_helpers", data)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.OutputDir, "tests", "helpers_test.go")
	return utils.WriteFile(filename, content)
}
