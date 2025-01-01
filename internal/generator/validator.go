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
	// Check required fields
	if err := v.validateRequiredFields(doc); err != nil {
		return err
	}

	// Validate paths
	if err := v.validatePaths(doc.Paths); err != nil {
		return err
	}

	// Validate components
	if err := v.validateComponents(doc.Components); err != nil {
		return err
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
