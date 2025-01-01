package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/logging"
	"github.com/chashtager/opensdkraft/internal/utils"
	"path/filepath"
	"strings"

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
	if schemas == nil {
		g.logger.Debug("No schemas to generate")
		return nil
	}

	modelsDir := filepath.Join(g.config.OutputDir, "models")
	g.logger.Debug("Creating models directory: %s", modelsDir)
	if err := utils.CreateDirectory(modelsDir); err != nil {
		g.logger.Error("Failed to create models directory: %v", err)
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	progress := g.logger.NewProgress(len(schemas), "Generating models")
	for name, schema := range schemas {
		g.logger.Debug("Generating model: %s", name)
		if err := g.generateModel(name, schema, modelsDir); err != nil {
			g.logger.Error("Failed to generate model %s: %v", name, err)
			return fmt.Errorf("failed to generate model %s: %w", name, err)
		}
		progress.Increment()
	}

	g.logger.Info("Successfully generated %d models", len(schemas))
	return nil
}

func (g *ModelGenerator) generateModel(name string, schema *openapi3.SchemaRef, outputDir string) error {
	modelData, err := g.prepareModelData(name, schema)
	if err != nil {
		return err
	}

	filename := filepath.Join(outputDir, strings.ToLower(name)+".go")

	content, err := g.templates.Execute("model", modelData)
	if err != nil {
		return err
	}

	return utils.WriteFile(filename, content)
}

type ModelData struct {
	Name       string
	Properties []PropertyData
	Imports    []string
	Config     *config.Config
}

type PropertyData struct {
	Name     string
	Type     string
	JSONName string
	Required bool
	Comment  string
}

func (g *ModelGenerator) prepareModelData(name string, schema *openapi3.SchemaRef) (*ModelData, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("invalid schema for %s", name)
	}

	modelData := &ModelData{
		Name:    g.typeMapper.ToGoName(name),
		Config:  g.config,
		Imports: make([]string, 0),
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

	return &PropertyData{
		Name:     g.typeMapper.ToGoName(name),
		Type:     goType,
		JSONName: name,
		Required: isRequired,
		Comment:  schema.Value.Description,
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
