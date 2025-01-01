// internal/parser/parser.go
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chashtager/opensdkraft/internal/errors"
	"github.com/getkin/kin-openapi/openapi3"
)

type Parser struct {
	loader *openapi3.Loader
}

func New() (*Parser, error) {
	return &Parser{
		loader: openapi3.NewLoader(),
	}, nil
}

func (p *Parser) ParseFile(filePath string) (*openapi3.T, error) {
	// Check file existence
	if _, err := os.Stat(filePath); err != nil {
		return nil, errors.FileSystemError(fmt.Errorf("file not found: %s", filePath))
	}

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return p.parseJSON(filePath)
	case ".yaml", ".yml":
		return p.parseYAML(filePath)
	default:
		return nil, errors.InvalidInput(fmt.Sprintf("unsupported file format: %s", ext))
	}
}

func (p *Parser) parseJSON(filePath string) (*openapi3.T, error) {
	doc, err := p.loader.LoadFromFile(filePath)
	if err != nil {
		return nil, errors.ParsingFailed(fmt.Errorf("failed to parse JSON file: %w", err))
	}
	return doc, nil
}

func (p *Parser) parseYAML(filePath string) (*openapi3.T, error) {
	doc, err := p.loader.LoadFromFile(filePath)
	if err != nil {
		return nil, errors.ParsingFailed(fmt.Errorf("failed to parse YAML file: %w", err))
	}
	return doc, nil
}

func (p *Parser) Validate(doc *openapi3.T) error {
	if err := doc.Validate(p.loader.Context); err != nil {
		return errors.ValidationFailed(err)
	}
	return nil
}

// GetOperations extracts all operations from the OpenAPI document
func (p *Parser) GetOperations(doc *openapi3.T) ([]*Operation, error) {
	var operations []*Operation

	for path, pathItem := range doc.Paths.Map() {
		pathOps, err := p.extractPathOperations(path, pathItem)
		if err != nil {
			return nil, err
		}
		operations = append(operations, pathOps...)
	}

	return operations, nil
}

// Operation represents an API operation
type Operation struct {
	ID          string
	Method      string
	Path        string
	Summary     string
	Description string
	Parameters  []*Parameter
	RequestBody *RequestBody
	Responses   map[string]*Response
	Security    []map[string][]string
	Tags        []string
}

type Parameter struct {
	Name        string
	In          string
	Required    bool
	Schema      *openapi3.SchemaRef
	Description string
}

type RequestBody struct {
	Required    bool
	ContentType string
	Schema      *openapi3.SchemaRef
	Description string
}

type Response struct {
	StatusCode  string
	ContentType string
	Schema      *openapi3.SchemaRef
	Description string
}

func (p *Parser) extractPathOperations(path string, pathItem *openapi3.PathItem) ([]*Operation, error) {
	var operations []*Operation

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

		operation, err := p.convertOperation(method, path, op)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeParsingFailed,
				fmt.Sprintf("failed to parse operation %s %s", method, path), err)
		}
		operations = append(operations, operation)
	}

	return operations, nil
}

func (p *Parser) convertOperation(method, path string, op *openapi3.Operation) (*Operation, error) {
	operation := &Operation{
		ID:          op.OperationID,
		Method:      method,
		Path:        path,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
		Security:    make([]map[string][]string, 0),
		Responses:   make(map[string]*Response),
	}

	// Convert parameters
	for _, param := range op.Parameters {
		parameter, err := p.convertParameter(param)
		if err != nil {
			return nil, err
		}
		operation.Parameters = append(operation.Parameters, parameter)
	}

	// Convert request body
	if op.RequestBody != nil {
		requestBody, err := p.convertRequestBody(op.RequestBody)
		if err != nil {
			return nil, err
		}
		operation.RequestBody = requestBody
	}

	// Convert responses
	for status, response := range op.Responses.Map() {
		resp, err := p.convertResponse(status, response)
		if err != nil {
			return nil, err
		}
		operation.Responses[status] = resp
	}

	// Convert security requirements
	for _, sec := range *op.Security {
		secMap := make(map[string][]string)
		for name, scopes := range sec {
			secMap[name] = scopes
		}
		operation.Security = append(operation.Security, secMap)
	}

	return operation, nil
}

func (p *Parser) convertParameter(param *openapi3.ParameterRef) (*Parameter, error) {
	if param.Value == nil {
		return nil, errors.InvalidInput("parameter value is nil")
	}

	return &Parameter{
		Name:        param.Value.Name,
		In:          param.Value.In,
		Required:    param.Value.Required,
		Schema:      param.Value.Schema,
		Description: param.Value.Description,
	}, nil
}

func (p *Parser) convertRequestBody(reqBody *openapi3.RequestBodyRef) (*RequestBody, error) {
	if reqBody.Value == nil {
		return nil, errors.InvalidInput("request body value is nil")
	}

	// Get the first content type (usually application/json)
	var contentType string
	var schema *openapi3.SchemaRef
	for ct, content := range reqBody.Value.Content {
		contentType = ct
		schema = content.Schema
		break
	}

	return &RequestBody{
		Required:    reqBody.Value.Required,
		ContentType: contentType,
		Schema:      schema,
		Description: reqBody.Value.Description,
	}, nil
}

func (p *Parser) convertResponse(status string, response *openapi3.ResponseRef) (*Response, error) {
	if response.Value == nil {
		return nil, errors.InvalidInput("response value is nil")
	}

	// Get the first content type (usually application/json)
	var contentType string
	var schema *openapi3.SchemaRef
	for ct, content := range response.Value.Content {
		contentType = ct
		schema = content.Schema
		break
	}

	return &Response{
		StatusCode:  status,
		ContentType: contentType,
		Schema:      schema,
		Description: *response.Value.Description,
	}, nil
}
