package generator

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type Validator struct {
	requiredFields []string
}

func NewValidator() *Validator {
	return &Validator{
		requiredFields: []string{"openapi", "info", "paths"},
	}
}

func (v *Validator) ValidateDocument(doc *openapi3.T) error {
	var validationErrors ValidationErrors

	// Validate required fields
	if err := v.validateRequiredFields(doc); err != nil {
		validationErrors.Add("Document", "root", err.Error())
	}

	// Validate paths
	for path, pathItem := range doc.Paths.Map() {
		if err := v.validatePathItem(path, pathItem); err != nil {
			validationErrors.Add("Path", path, err.Error())
		}
	}

	// Validate components
	if doc.Components != nil {
		// Validate schemas
		for name, schema := range doc.Components.Schemas {
			if errs := v.validateSchema(schema); len(errs) > 0 {
				validationErrors.Add("Schema", name, errs...)
			}
		}

		// Validate responses
		for name, response := range doc.Components.Responses {
			if err := v.validateResponse(name, response); err != nil {
				validationErrors.Add("Response", name, err.Error())
			}
		}
	}

	if len(validationErrors.Errors) > 0 {
		return &validationErrors
	}
	return nil
}

func (v *Validator) validateSchema(schema *openapi3.SchemaRef) []string {
	var errors []string

	if schema == nil || schema.Value == nil {
		return []string{"schema is nil"}
	}

	// Type validation
	if len(schema.Value.Type.Slice()) == 0 {
		errors = append(errors, "missing type")
	}

	// Properties validation for objects
	if schema.Value.Type.Slice()[0] == "object" {
		if schema.Value.Properties == nil && schema.Value.AdditionalProperties.Schema == nil {
			errors = append(errors, "object schema must define either properties or additionalProperties")
		}

		for propName, prop := range schema.Value.Properties {
			if prop == nil || prop.Value == nil {
				errors = append(errors, fmt.Sprintf("property %s is nil", propName))
				continue
			}

			if propErrs := v.validateSchema(prop); len(propErrs) > 0 {
				for _, err := range propErrs {
					errors = append(errors, fmt.Sprintf("property %s: %s", propName, err))
				}
			}
		}
	}

	// Array validation
	if schema.Value.Type.Slice()[0] == "array" {
		if schema.Value.Items == nil {
			errors = append(errors, "array schema must define items")
		} else if itemErrors := v.validateSchema(schema.Value.Items); len(itemErrors) > 0 {
			errors = append(errors, itemErrors...)
		}
	}

	return errors
}

func (v *Validator) validateResponse(name string, response *openapi3.ResponseRef) error {
	if response == nil || response.Value == nil {
		return fmt.Errorf("response is nil")
	}

	for mediaType, content := range response.Value.Content {
		if content.Schema == nil {
			return fmt.Errorf("missing schema for media type %s", mediaType)
		}

		if errs := v.validateSchema(content.Schema); len(errs) > 0 {
			return fmt.Errorf("invalid schema for media type %s: %s", mediaType, strings.Join(errs, ", "))
		}
	}

	return nil
}

func (v *Validator) validateRequiredFields(doc *openapi3.T) error {
	if doc.OpenAPI == "" {
		return fmt.Errorf("OpenAPI version is required")
	}

	if !strings.HasPrefix(doc.OpenAPI, "3.") {
		return fmt.Errorf("only OpenAPI 3.x is supported, got %s", doc.OpenAPI)
	}

	if doc.Info == nil {
		return fmt.Errorf("info section is required")
	}

	if doc.Paths == nil {
		return fmt.Errorf("paths section is required")
	}

	return nil
}

func (v *Validator) validatePaths(paths *openapi3.Paths) error {
	if paths == nil {
		return fmt.Errorf("paths cannot be nil")
	}

	// Get all paths using methods from openapi3.Paths
	for path, pathItem := range paths.Map() {
		if pathItem == nil {
			return fmt.Errorf("path %s has no operations", path)
		}

		if err := v.validatePathItem(path, pathItem); err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validatePathItem(path string, pathItem *openapi3.PathItem) error {
	// Check each operation type
	if pathItem.Get != nil {
		if err := v.validateOperation(path, "GET", pathItem.Get); err != nil {
			return err
		}
	}
	if pathItem.Post != nil {
		if err := v.validateOperation(path, "POST", pathItem.Post); err != nil {
			return err
		}
	}
	if pathItem.Put != nil {
		if err := v.validateOperation(path, "PUT", pathItem.Put); err != nil {
			return err
		}
	}
	if pathItem.Delete != nil {
		if err := v.validateOperation(path, "DELETE", pathItem.Delete); err != nil {
			return err
		}
	}
	if pathItem.Patch != nil {
		if err := v.validateOperation(path, "PATCH", pathItem.Patch); err != nil {
			return err
		}
	}
	if pathItem.Head != nil {
		if err := v.validateOperation(path, "HEAD", pathItem.Head); err != nil {
			return err
		}
	}
	if pathItem.Options != nil {
		if err := v.validateOperation(path, "OPTIONS", pathItem.Options); err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateOperation(path, method string, op *openapi3.Operation) error {
	if op.Responses == nil {
		return fmt.Errorf("path %s %s has no responses defined", path, method)
	}

	return nil
}

func (v *Validator) validateComponents(components *openapi3.Components) error {
	if components == nil {
		return nil
	}

	// Validate schemas
	for name, schema := range components.Schemas {
		if schema == nil || schema.Value == nil {
			return fmt.Errorf("invalid schema definition for %s", name)
		}
	}

	return nil
}

// ValidationError holds validation details
type ValidationError struct {
	Category string   // e.g., "Model", "Operation", "Schema"
	Path     string   // e.g., "Pet.name", "/pet/{petId}"
	Errors   []string // List of specific error messages
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s validation failed for %s:\n%s",
		ve.Category,
		ve.Path,
		strings.Join(ve.Errors, "\n  - "))
}

// ValidationErrors collects multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError
}

func (ve *ValidationErrors) Error() string {
	var msgs []string
	for _, err := range ve.Errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "\n")
}

func (ve *ValidationErrors) Add(category, path string, errors ...string) {
	ve.Errors = append(ve.Errors, ValidationError{
		Category: category,
		Path:     path,
		Errors:   errors,
	})
}
