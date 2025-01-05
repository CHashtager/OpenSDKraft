package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/errors"
	"github.com/chashtager/opensdkraft/internal/logging"
	"github.com/chashtager/opensdkraft/internal/utils"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type OperationGenerator struct {
	config     *config.Config
	templates  *TemplateEngine
	operations []*Operation
	typeMapper *TypeMapper
	logger     *logging.Logger
}

type Operation struct {
	Name           string
	Method         string
	Path           string
	Description    string
	RequestType    string
	ResponseType   string
	Parameters     []Parameter
	RequestBody    *RequestBody
	Responses      map[string]Response
	Authentication bool
	HasQueryParams bool
	HasPathParams  bool
	HasContext     bool
	ZeroValue      string
	ExampleValue   string
}

type Parameter struct {
	Name         string
	GoName       string
	Type         string
	Location     string // path, query, header
	Required     bool
	Description  string
	JSONName     string
	Validate     string
	Validation   []string
	ZeroValue    string
	ExampleValue string
}

type RequestBody struct {
	Type         string
	Required     bool
	MediaType    string
	Description  string
	ExampleValue string
}

type Response struct {
	StatusCode  string
	Type        string
	MediaType   string
	Description string
}

func NewOperationGenerator(config *config.Config, templates *TemplateEngine, logger *logging.Logger) *OperationGenerator {
	return &OperationGenerator{
		config:     config,
		templates:  templates,
		operations: make([]*Operation, 0),
		typeMapper: NewTypeMapper(config),
		logger:     logger,
	}
}

func (g *OperationGenerator) Generate(paths *openapi3.Paths) error {
	if paths == nil {
		g.logger.Debug("No paths to generate")
		return errors.InvalidInput("paths cannot be nil")
	}

	// Create operations directory
	operationsDir := filepath.Join(g.config.OutputDir, "operations")
	g.logger.Debug("Creating operations directory: %s", operationsDir)
	if err := utils.CreateDirectory(operationsDir); err != nil {
		g.logger.Error("Failed to create operations directory: %v", err)
		return errors.FileSystemError(err)
	}

	// Generate operations
	pathMap := paths.Map()
	progress := g.logger.NewProgress(len(pathMap), "Generating operations")
	for path, pathItem := range pathMap {
		g.logger.Debug("Processing path: %s", path)
		if err := g.generatePathOperations(path, pathItem, operationsDir); err != nil {
			g.logger.Error("Failed to generate operations for path %s: %v", path, err)
			return errors.Wrap(errors.ErrCodeGenerationFailed,
				fmt.Sprintf("failed to generate operations for path %s", path), err)
		}
		progress.Increment()
	}

	// Generate client file with all operations
	g.logger.Info("Generating client file")
	if err := g.generateClientFile(); err != nil {
		g.logger.Error("Failed to generate client file: %v", err)
		return errors.Wrap(errors.ErrCodeGenerationFailed,
			"failed to generate client file", err)
	}

	g.logger.Info("Successfully generated %d operations", len(g.operations))
	return nil
}

func (g *OperationGenerator) generatePathOperations(path string, pathItem *openapi3.PathItem, outputDir string) error {
	methods := map[string]*openapi3.Operation{
		"GET":     pathItem.Get,
		"POST":    pathItem.Post,
		"PUT":     pathItem.Put,
		"DELETE":  pathItem.Delete,
		"PATCH":   pathItem.Patch,
		"HEAD":    pathItem.Head,
		"OPTIONS": pathItem.Options,
	}

	for method, op := range methods {
		if op == nil {
			continue
		}

		operation, err := g.parseOperation(method, path, op)
		if err != nil {
			return err
		}
		g.operations = append(g.operations, operation)

		// Generate operation file
		if err := g.generateOperationFile(operation, outputDir); err != nil {
			return err
		}
	}

	return nil
}

func (g *OperationGenerator) parseOperation(method, path string, op *openapi3.Operation) (*Operation, error) {
	operation := &Operation{
		Name:           g.generateOperationName(method, path, op),
		Method:         method,
		Path:           path,
		Description:    op.Description,
		Parameters:     make([]Parameter, 0),
		Responses:      make(map[string]Response),
		Authentication: op.Security != nil,
		HasContext:     g.config.Generator.ClientOptions.UseContext,
	}

	// Parse parameters
	for _, paramRef := range op.Parameters {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		parameter, err := g.parseParameter(paramRef)
		if err != nil {
			return nil, err
		}
		operation.Parameters = append(operation.Parameters, *parameter)
		// Set parameter type flags
		switch paramRef.Value.In {
		case "query":
			operation.HasQueryParams = true
		case "path":
			operation.HasPathParams = true
		}
	}

	// Parse request body
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		requestBody, err := g.parseRequestBody(op.RequestBody)
		if err != nil {
			return nil, err
		}
		operation.RequestBody = requestBody
	}

	// Parse responses
	for status, responseRef := range op.Responses.Map() {
		if responseRef == nil || responseRef.Value == nil {
			continue
		}
		resp, err := g.parseResponse(status, responseRef)
		if err != nil {
			return nil, err
		}
		operation.Responses[status] = *resp
	}

	return operation, nil
}

func (g *OperationGenerator) parseParameter(paramRef *openapi3.ParameterRef) (*Parameter, error) {
	if paramRef.Value == nil {
		return nil, errors.InvalidInput("parameter value is nil")
	}

	param := paramRef.Value
	goType, _ := g.typeMapper.ToGoType(param.Schema)
	validate := g.generateParamValidation(param)

	return &Parameter{
		Name:         param.Name,
		GoName:       g.typeMapper.ToGoName(param.Name),
		Type:         goType,
		Location:     param.In,
		Required:     param.Required,
		Description:  param.Description,
		JSONName:     param.Name,
		Validate:     validate,
		ZeroValue:    g.getZeroValue(goType),
		ExampleValue: g.getExampleValue(param.Schema),
	}, nil
}

func (g *OperationGenerator) parseRequestBody(bodyRef *openapi3.RequestBodyRef) (*RequestBody, error) {
	if bodyRef.Value == nil {
		return nil, errors.InvalidInput("request body value is nil")
	}

	body := bodyRef.Value
	// Get the first content type and its schema
	var mediaType string
	var schema *openapi3.SchemaRef
	for mt, content := range body.Content {
		mediaType = mt
		schema = content.Schema
		break
	}

	if schema == nil {
		return nil, errors.InvalidInput("request body schema is nil")
	}

	goType, _ := g.typeMapper.ToGoType(schema)
	exampleValue := g.generateExampleValue(schema)

	return &RequestBody{
		Type:         goType,
		Required:     body.Required,
		MediaType:    mediaType,
		Description:  body.Description,
		ExampleValue: exampleValue,
	}, nil
}

func (g *OperationGenerator) generateExampleValue(schema *openapi3.SchemaRef) string {
	if schema == nil || schema.Value == nil {
		return "nil"
	}

	// If schema has an example, use it
	if schema.Value.Example != nil {
		return fmt.Sprintf("%#v", schema.Value.Example)
	}

	// Generate example based on schema type
	schemaType := ""
	if len(schema.Value.Type.Slice()) > 0 {
		schemaType = schema.Value.Type.Slice()[0]
	}

	switch schemaType {
	case "array":
		if schema.Value.Items != nil {
			itemExample := g.generateExampleValue(schema.Value.Items)
			itemType, _ := g.typeMapper.ToGoType(schema.Value.Items)
			return fmt.Sprintf("[]%s{%s}", itemType, itemExample)
		}
		return "[]interface{}{}"

	case "object":
		if ref := schema.Ref; ref != "" {
			// Reference to another schema
			parts := strings.Split(ref, "/")
			typeName := parts[len(parts)-1]
			return fmt.Sprintf("&%s{}", g.typeMapper.ToGoName(typeName))
		}
		return "map[string]interface{}{}"

	case "string":
		if schema.Value.Enum != nil && len(schema.Value.Enum) > 0 {
			return fmt.Sprintf("%#v", schema.Value.Enum[0])
		}
		return `"example"`

	case "integer":
		if schema.Value.Format == "int64" {
			return "int64(123)"
		}
		return "42"

	case "number":
		if schema.Value.Format == "float" {
			return "float32(1.23)"
		}
		return "1.23"

	case "boolean":
		return "true"

	default:
		return "nil"
	}
}

// Helper function to generate example values for request/response
func (g *OperationGenerator) generateExampleResponse(responses map[string]*openapi3.ResponseRef) string {
	// Look for 200 or 201 response first
	for _, statusCode := range []string{"200", "201"} {
		if resp, ok := responses[statusCode]; ok {
			for _, content := range resp.Value.Content {
				if content.Schema != nil {
					return g.generateExampleValue(content.Schema)
				}
			}
		}
	}

	// If no successful response found, try any response
	for _, resp := range responses {
		if resp.Value == nil {
			continue
		}
		for _, content := range resp.Value.Content {
			if content.Schema != nil {
				return g.generateExampleValue(content.Schema)
			}
		}
	}

	return "nil"
}
func (g *OperationGenerator) parseResponse(status string, responseRef *openapi3.ResponseRef) (*Response, error) {
	if responseRef.Value == nil {
		return &Response{
			StatusCode:  status,
			Type:        "error",
			MediaType:   "application/json",
			Description: "Error response",
		}, nil
	}

	resp := responseRef.Value

	// Default response type if no content is specified
	responseType := "void"
	mediaType := "application/json"

	if resp.Content != nil && len(resp.Content) > 0 {
		// Find the first available content type (prefer JSON)
		for mt, content := range resp.Content {
			if content.Schema != nil {
				goType, _ := g.typeMapper.ToGoType(content.Schema)
				return &Response{
					StatusCode:  status,
					Type:        goType,
					MediaType:   mt,
					Description: *resp.Description,
				}, nil
			}
			mediaType = mt
		}
	}

	// Return a default response if no schema is found
	return &Response{
		StatusCode:  status,
		Type:        responseType,
		MediaType:   mediaType,
		Description: *resp.Description,
	}, nil
}

func (g *OperationGenerator) generateAPIClient(operations []*Operation) error {
	data := struct {
		PackageName string
		Operations  []*Operation
		Config      *config.Config
	}{
		PackageName: g.config.PackageName,
		Operations:  operations,
		Config:      g.config,
	}

	content, err := g.templates.Execute("client", data)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.OutputDir, "client.go")
	return utils.WriteFile(filename, content)
}

func (g *OperationGenerator) generateOperationFile(operation *Operation, outputDir string) error {
	data := struct {
		Operation   *Operation
		PackageName string
		Config      *config.Config
	}{
		Operation:   operation,
		PackageName: g.config.PackageName,
		Config:      g.config,
	}

	content, err := g.templates.Execute("operation", data)
	if err != nil {
		return err
	}

	filename := filepath.Join(outputDir, "operations", strings.ToLower(operation.Name)+".go")
	return utils.WriteFile(filename, content)
}

func (g *OperationGenerator) generateOperationName(method, path string, op *openapi3.Operation) string {
	if op.OperationID != "" {
		return utils.ToCamelCase(op.OperationID)
	}

	// Generate name from method and path
	parts := strings.Split(path, "/")
	var nameParts []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Remove path parameters
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = "By" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		}
		nameParts = append(nameParts, part)
	}

	return utils.ToCamelCase(method + strings.Join(nameParts, ""))
}

func (g *OperationGenerator) generateClientFile() error {
	g.logger.Debug("Generating client file")

	data := struct {
		PackageName string
		Operations  []*Operation
		Config      *config.Config
	}{
		PackageName: g.config.PackageName,
		Operations:  g.operations,
		Config:      g.config,
	}

	content, err := g.templates.Execute("client", data)
	if err != nil {
		return errors.Wrap(errors.ErrCodeTemplateError,
			"failed to execute client template", err)
	}

	filename := filepath.Join(g.config.OutputDir, "client.go")
	g.logger.Debug("Writing client file to: %s", filename)

	if err := utils.WriteFile(filename, content); err != nil {
		return errors.Wrap(errors.ErrCodeFileSystemError,
			"failed to write client file", err)
	}

	return nil
}

// GetOperations returns the list of generated operations
func (g *OperationGenerator) GetOperations() []*Operation {
	return g.operations
}

func (g *OperationGenerator) generateParamValidation(param *openapi3.Parameter) string {
	var validations []string

	if param.Required {
		validations = append(validations, "required")
	}

	if param.Schema != nil && param.Schema.Value != nil {
		schema := param.Schema.Value

		// Add schema-based validations
		if schema.MinLength != 0 {
			validations = append(validations, fmt.Sprintf("min=%d", schema.MinLength))
		}
		if schema.MaxLength != nil {
			validations = append(validations, fmt.Sprintf("max=%d", *schema.MaxLength))
		}
		if schema.Pattern != "" {
			validations = append(validations, fmt.Sprintf("regexp=%s", schema.Pattern))
		}
		if schema.Enum != nil {
			enums := make([]string, len(schema.Enum))
			for i, v := range schema.Enum {
				enums[i] = fmt.Sprintf("%v", v)
			}
			validations = append(validations, fmt.Sprintf("oneof=%s", strings.Join(enums, " ")))
		}
	}

	if len(validations) > 0 {
		return strings.Join(validations, ",")
	}
	return ""
}

func (g *OperationGenerator) getZeroValue(goType string) string {
	switch goType {
	case "string":
		return `""`
	case "int", "int32", "int64", "float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

func (g *OperationGenerator) getExampleValue(schema *openapi3.SchemaRef) string {
	if schema != nil && schema.Value != nil {
		if schema.Value.Example != nil {
			return fmt.Sprintf("%#v", schema.Value.Example)
		}

		switch schema.Value.Type.Slice()[0] {
		case "string":
			return `"example"`
		case "integer":
			return "1"
		case "number":
			return "1.0"
		case "boolean":
			return "true"
		}
	}
	return "nil"
}
