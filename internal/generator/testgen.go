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

type TestData struct {
	Operation    *Operation
	MockData     map[string]interface{}
	Expectations []TestExpectation
}

type TestExpectation struct {
	Name        string
	Request     interface{}
	Response    interface{}
	StatusCode  int
	ErrorCase   bool
	Description string
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
		if err := g.generateOperationTests(op); err != nil {
			return fmt.Errorf("failed to generate tests for operation %s: %w", op.Name, err)
		}
	}

	// Generate test helpers
	if err := g.generateTestHelpers(); err != nil {
		return fmt.Errorf("failed to generate test helpers: %w", err)
	}

	return nil
}

func (g *TestGenerator) generateOperationTests(op *Operation) error {
	testData, err := g.prepareTestData(op)
	if err != nil {
		return err
	}

	data := struct {
		TestData    *TestData
		PackageName string
		Config      *config.Config
	}{
		TestData:    testData,
		PackageName: g.config.PackageName,
		Config:      g.config,
	}

	content, err := g.templates.Execute("operation_test", data)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.OutputDir, "tests", strings.ToLower(op.Name)+"_test.go")
	return utils.WriteFile(filename, content)
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

func (g *TestGenerator) prepareTestData(op *Operation) (*TestData, error) {
	testData := &TestData{
		Operation: op,
		MockData:  make(map[string]interface{}),
		Expectations: []TestExpectation{
			// Success case
			{
				Name:        "Success",
				StatusCode:  200,
				ErrorCase:   false,
				Description: fmt.Sprintf("Test successful %s operation", strings.ToLower(op.Method)),
			},
			// Error cases
			{
				Name:        "Unauthorized",
				StatusCode:  401,
				ErrorCase:   true,
				Description: "Test unauthorized access",
			},
			{
				Name:        "NotFound",
				StatusCode:  404,
				ErrorCase:   true,
				Description: "Test resource not found",
			},
		},
	}

	// Generate mock data for request and response
	if err := g.generateMockData(op, testData); err != nil {
		return nil, err
	}

	return testData, nil
}

func (g *TestGenerator) generateMockData(op *Operation, testData *TestData) error {
	// Generate mock request data
	if op.RequestBody != nil {
		mockReq, err := g.generateMockType(op.RequestBody.Type)
		if err != nil {
			return err
		}
		testData.MockData["request"] = mockReq
	}

	// Generate mock response data
	for _, resp := range op.Responses {
		mockResp, err := g.generateMockType(resp.Type)
		if err != nil {
			return err
		}
		testData.MockData[resp.StatusCode] = mockResp
	}

	return nil
}

func (g *TestGenerator) generateMockType(typeName string) (interface{}, error) {
	// Implementation depends on your type system
	// This is a simplified version
	switch typeName {
	case "string":
		return "mock_string", nil
	case "int":
		return 123, nil
	case "bool":
		return true, nil
	default:
		return map[string]interface{}{
			"id":   "mock_id",
			"name": "mock_name",
		}, nil
	}
}
