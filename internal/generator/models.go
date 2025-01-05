package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/logging"
	"github.com/chashtager/opensdkraft/internal/utils"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

type ModelGenerator struct {
	config     *config.Config
	templates  *TemplateEngine
	typeMapper *TypeMapper
	logger     *logging.Logger
}

func NewModelGenerator(config *config.Config, templates *TemplateEngine, logger *logging.Logger) *ModelGenerator {
	return &ModelGenerator{
		config:     config,
		templates:  templates,
		typeMapper: NewTypeMapper(config),
		logger:     logger,
	}
}

type TypeMapper struct {
	config     *config.Config
	knownTypes map[string]string
}

func NewTypeMapper(config *config.Config) *TypeMapper {
	tm := &TypeMapper{
		config:     config,
		knownTypes: make(map[string]string),
	}
	tm.initializeKnownTypes()
	return tm
}

func (tm *TypeMapper) initializeKnownTypes() {
	// Basic types
	tm.knownTypes["string"] = "string"
	tm.knownTypes["number"] = "float64"
	tm.knownTypes["integer"] = "int"
	tm.knownTypes["boolean"] = "bool"
	tm.knownTypes["array"] = "[]interface{}"
	tm.knownTypes["object"] = "map[string]interface{}"

	// Common formats
	tm.knownTypes["int32"] = "int32"
	tm.knownTypes["int64"] = "int64"
	tm.knownTypes["float"] = "float32"
	tm.knownTypes["double"] = "float64"
	tm.knownTypes["byte"] = "[]byte"
	tm.knownTypes["binary"] = "[]byte"
	tm.knownTypes["date"] = "time.Time"
	tm.knownTypes["date-time"] = "time.Time"
	tm.knownTypes["password"] = "string"
	tm.knownTypes["email"] = "string"
	tm.knownTypes["uuid"] = "string"
	tm.knownTypes["uri"] = "string"
	tm.knownTypes["hostname"] = "string"
	tm.knownTypes["ipv4"] = "string"
	tm.knownTypes["ipv6"] = "string"
}

func (g *ModelGenerator) Generate(schemas openapi3.Schemas) error {
	var validationErrors ValidationErrors

	for name, schema := range schemas {
		if err := g.generateModel(name, schema); err != nil {
			if valErr, ok := err.(*ValidationError); ok {
				validationErrors.Errors = append(validationErrors.Errors, *valErr)
			} else {
				validationErrors.Add("Model", name, err.Error())
			}
		}
	}

	if len(validationErrors.Errors) > 0 {
		return &validationErrors
	}

	return nil
}

func (g *ModelGenerator) generateModel(name string, schema *openapi3.SchemaRef) error {
	modelData, err := g.prepareModelData(name, schema)
	if err != nil {
		return &ValidationError{
			Category: "Model",
			Path:     name,
			Errors:   []string{err.Error()},
		}
	}

	// Validate the model data
	if errs := g.validateModelData(modelData); len(errs) > 0 {
		return &ValidationError{
			Category: "Model",
			Path:     name,
			Errors:   errs,
		}
	}

	// Generate the model file
	content, err := g.templates.Execute("model", modelData)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	filename := filepath.Join(g.config.OutputDir, "models", strings.ToLower(name)+".go")
	if err := utils.WriteFile(filename, content); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (g *ModelGenerator) validateModelData(data *ModelData) []string {
	var errors []string

	if data.Name == "" {
		errors = append(errors, "model name is required")
	}

	if !isValidGoIdentifier(data.Name) {
		errors = append(errors, fmt.Sprintf("invalid model name: %s", data.Name))
	}

	for _, prop := range data.Properties {
		if !isValidGoIdentifier(prop.Name) {
			errors = append(errors, fmt.Sprintf("invalid property name: %s", prop.Name))
		}

		if prop.Type == "" {
			errors = append(errors, fmt.Sprintf("property %s has no type", prop.Name))
		}
	}

	return errors
}

func isValidGoIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		if i == 0 {
			if !unicode.IsLetter(c) && c != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
				return false
			}
		}
	}
	return true
}

//func (g *ModelGenerator) generateModel(name string, schema *openapi3.SchemaRef, outputDir string) error {
//	modelData, err := g.prepareModelData(name, schema)
//	if err != nil {
//		return err
//	}
//
//	filename := filepath.Join(outputDir, strings.ToLower(name)+".go")
//
//	content, err := g.templates.Execute("model", modelData)
//	if err != nil {
//		return err
//	}
//
//	return utils.WriteFile(filename, content)
//}

type ModelData struct {
	Name        string
	PackageName string
	Properties  []PropertyData
	Imports     []string
	Config      *config.Config
	Description string
}

type PropertyData struct {
	Name         string
	Type         string
	JSONName     string
	Required     bool
	Description  string
	Validate     string
	Validation   []string
	ZeroValue    string
	ExampleValue string
}

func (g *ModelGenerator) prepareModelData(name string, schema *openapi3.SchemaRef) (*ModelData, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("invalid schema for %s", name)
	}

	modelData := &ModelData{
		Name:        g.typeMapper.ToGoName(name),
		PackageName: g.config.PackageName,
		Config:      g.config,
		Imports:     make([]string, 0),
		Description: schema.Value.Description,
	}

	seenImports := make(map[string]bool)

	for propName, propSchema := range schema.Value.Properties {
		propData, imports, err := g.preparePropertyData(propName, propSchema, schema.Value.Required)
		if err != nil {
			return nil, err
		}

		modelData.Properties = append(modelData.Properties, *propData)

		for _, imp := range imports {
			if !seenImports[imp] {
				modelData.Imports = append(modelData.Imports, imp)
				seenImports[imp] = true
			}
		}
	}

	return modelData, nil
}

func (g *ModelGenerator) preparePropertyData(name string, schema *openapi3.SchemaRef, required []string) (*PropertyData, []string, error) {
	if schema == nil || schema.Value == nil {
		return nil, nil, fmt.Errorf("invalid property schema for %s", name)
	}

	goType, imports := g.typeMapper.ToGoType(schema)
	isRequired := utils.StringContains(required, name)
	validate := g.generateValidationTag(schema, isRequired)

	return &PropertyData{
		Name:         g.typeMapper.ToGoName(name),
		Type:         goType,
		JSONName:     name,
		Required:     isRequired,
		Description:  schema.Value.Description,
		Validate:     validate,
		ZeroValue:    g.getZeroValue(goType),
		ExampleValue: g.getExampleValue(schema),
	}, imports, nil
}

func (tm *TypeMapper) ToGoType(schema *openapi3.SchemaRef) (string, []string) {
	if schema == nil || schema.Value == nil {
		return "interface{}", nil
	}

	var imports []string
	// Since Type is []string, we need to check if it's not empty and take the first type
	schemaType := ""
	if len(*schema.Value.Type) > 0 {
		schemaType = schema.Value.Type.Slice()[0]
	}
	format := schema.Value.Format

	// Handle special formats first
	if format != "" {
		if typ, ok := tm.knownTypes[format]; ok {
			if format == "date" || format == "date-time" {
				imports = append(imports, "time")
			}
			return typ, imports
		}
	}

	// Handle regular types based on schema type
	switch schemaType {
	case "array":
		if schema.Value.Items != nil {
			itemType, itemImports := tm.ToGoType(schema.Value.Items)
			imports = append(imports, itemImports...)
			return "[]" + itemType, imports
		}
		return "[]interface{}", nil

	case "object":
		// Check for AdditionalProperties
		if schema.Value.AdditionalProperties.Has != nil &&
			schema.Value.AdditionalProperties.Schema != nil {
			valueType, valueImports := tm.ToGoType(schema.Value.AdditionalProperties.Schema)
			imports = append(imports, valueImports...)
			return "map[string]" + valueType, imports
		}
		// Handle specific object properties
		if len(schema.Value.Properties) > 0 {
			// This is a structured object, should be handled by model generation
			return "struct", nil
		}
		return "map[string]interface{}", nil

	case "string":
		if typ, ok := tm.knownTypes["string"]; ok {
			return typ, imports
		}
		return "string", nil

	case "number":
		if format == "float" {
			return "float32", nil
		}
		return "float64", nil

	case "integer":
		if format == "int64" {
			return "int64", nil
		}
		return "int", nil

	case "boolean":
		return "bool", nil

	default:
		if typ, ok := tm.knownTypes[schemaType]; ok {
			return typ, imports
		}
		return "interface{}", nil
	}
}

func (tm *TypeMapper) ToGoName(name string) string {
	// Convert to PascalCase
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r)
	})

	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}

	return strings.Join(words, "")
}

func (g *ModelGenerator) getZeroValue(goType string) string {
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

func (g *ModelGenerator) getExampleValue(schema *openapi3.SchemaRef) string {
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
	default:
		return "nil"
	}
}

func (g *ModelGenerator) generateValidationTag(schema *openapi3.SchemaRef, required bool) string {
	var validations []string

	if required {
		validations = append(validations, "required")
	}

	// Add other validations based on schema
	if schema.Value.MinLength != 0 {
		validations = append(validations, fmt.Sprintf("min=%d", schema.Value.MinLength))
	}
	if schema.Value.MaxLength != nil {
		validations = append(validations, fmt.Sprintf("max=%d", *schema.Value.MaxLength))
	}
	if schema.Value.Pattern != "" {
		validations = append(validations, fmt.Sprintf("regexp=%s", schema.Value.Pattern))
	}
	if schema.Value.Enum != nil {
		enums := make([]string, len(schema.Value.Enum))
		for i, v := range schema.Value.Enum {
			enums[i] = fmt.Sprintf("%v", v)
		}
		validations = append(validations, fmt.Sprintf("oneof=%s", strings.Join(enums, " ")))
	}

	if len(validations) > 0 {
		return strings.Join(validations, ",")
	}
	return ""
}
