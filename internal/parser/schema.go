package parser

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type SchemaParser struct {
	doc     *openapi3.T
	schemas map[string]*Schema
}

type Schema struct {
	Name        string
	Description string
	Type        string
	Format      string
	Required    []string
	Properties  map[string]*SchemaProperty
	Enum        []interface{}
	IsArray     bool
	ItemSchema  *Schema
	Imports     []string
}

type SchemaProperty struct {
	Name        string
	Type        string
	Format      string
	Description string
	Required    bool
	IsPointer   bool
	IsArray     bool
	ItemSchema  *Schema
}

func NewSchemaParser(doc *openapi3.T) *SchemaParser {
	return &SchemaParser{
		doc:     doc,
		schemas: make(map[string]*Schema),
	}
}

func (sp *SchemaParser) ParseSchemas() error {
	if sp.doc.Components == nil || sp.doc.Components.Schemas == nil {
		return nil
	}

	for name, schemaRef := range sp.doc.Components.Schemas {
		schema, err := sp.parseSchemaRef(name, schemaRef)
		if err != nil {
			return fmt.Errorf("failed to parse schema %s: %w", name, err)
		}
		sp.schemas[name] = schema
	}

	return nil
}

func (sp *SchemaParser) parseSchemaRef(name string, schemaRef *openapi3.SchemaRef) (*Schema, error) {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil, fmt.Errorf("invalid schema reference for %s", name)
	}

	schema := &Schema{
		Name:        name,
		Description: schemaRef.Value.Description,
		Type:        schemaRef.Value.Type.Slice()[0],
		Format:      schemaRef.Value.Format,
		Required:    schemaRef.Value.Required,
		Properties:  make(map[string]*SchemaProperty),
		Enum:        schemaRef.Value.Enum,
	}

	if schema.Type == "array" && schemaRef.Value.Items != nil {
		schema.IsArray = true
		itemSchema, err := sp.parseSchemaRef(name+"Item", schemaRef.Value.Items)
		if err != nil {
			return nil, err
		}
		schema.ItemSchema = itemSchema
	}

	for propName, propRef := range schemaRef.Value.Properties {
		prop, err := sp.parseProperty(propName, propRef, schema.Required)
		if err != nil {
			return nil, err
		}
		schema.Properties[propName] = prop
	}

	schema.Imports = sp.determineImports(schema)
	return schema, nil
}

func (sp *SchemaParser) parseProperty(name string, propRef *openapi3.SchemaRef, required []string) (*SchemaProperty, error) {
	if propRef == nil || propRef.Value == nil {
		return nil, fmt.Errorf("invalid property reference for %s", name)
	}

	prop := &SchemaProperty{
		Name:        name,
		Type:        propRef.Value.Type.Slice()[0],
		Format:      propRef.Value.Format,
		Description: propRef.Value.Description,
		Required:    contains(required, name),
		IsPointer:   !contains(required, name),
	}

	if prop.Type == "array" && propRef.Value.Items != nil {
		prop.IsArray = true
		itemSchema, err := sp.parseSchemaRef(name+"Item", propRef.Value.Items)
		if err != nil {
			return nil, err
		}
		prop.ItemSchema = itemSchema
	}

	return prop, nil
}

func (sp *SchemaParser) determineImports(schema *Schema) []string {
	imports := make(map[string]bool)

	// Add time import if needed
	if schema.Type == "string" && schema.Format == "date-time" {
		imports["time"] = true
	}

	// Check properties for imports
	for _, prop := range schema.Properties {
		if prop.Type == "string" && prop.Format == "date-time" {
			imports["time"] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
