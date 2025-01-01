package generator

import (
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"unicode"

	"github.com/chashtager/opensdkraft/internal/errors"
)

type CodeValidator struct {
	config *config.Config
}

func NewCodeValidator(config *config.Config) *CodeValidator {
	return &CodeValidator{
		config: config,
	}
}

func (v *CodeValidator) ValidateGoCode(filename string, content []byte) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, content, parser.AllErrors)
	if err != nil {
		return errors.Wrap(errors.ErrCodeValidationFailed,
			fmt.Sprintf("failed to parse Go file %s", filename), err)
	}

	var validationErrors []string

	// Check package name
	if !v.isValidPackageName(file.Name.Name) {
		validationErrors = append(validationErrors,
			fmt.Sprintf("invalid package name: %s", file.Name.Name))
	}

	// Check imports
	if err := v.validateImports(file.Imports); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Check declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if err := v.validateTypeSpec(x); err != nil {
				validationErrors = append(validationErrors, err.Error())
			}
		case *ast.FuncDecl:
			if err := v.validateFuncDecl(x); err != nil {
				validationErrors = append(validationErrors, err.Error())
			}
		}
		return true
	})

	if len(validationErrors) > 0 {
		return errors.InvalidInput(fmt.Sprintf(
			"code validation failed:\n%s", strings.Join(validationErrors, "\n")))
	}

	return nil
}

func (v *CodeValidator) isValidPackageName(name string) bool {
	// Package names should be all lowercase and a single word
	return name == strings.ToLower(name) && !strings.Contains(name, "_")
}

func (v *CodeValidator) validateImports(imports []*ast.ImportSpec) error {
	var validationErrors []string
	seenImports := make(map[string]bool)

	for _, imp := range imports {
		path := strings.Trim(imp.Path.Value, `"`)

		// Check for duplicate imports
		if seenImports[path] {
			validationErrors = append(validationErrors,
				fmt.Sprintf("duplicate import: %s", path))
			continue
		}
		seenImports[path] = true

		// Check import path format
		if !v.isValidImportPath(path) {
			validationErrors = append(validationErrors,
				fmt.Sprintf("invalid import path: %s", path))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("import validation failed:\n%s",
			strings.Join(validationErrors, "\n"))
	}

	return nil
}

func (v *CodeValidator) isValidImportPath(path string) bool {
	// Basic import path validation
	return !strings.Contains(path, "..") &&
		!strings.Contains(path, "\\") &&
		!strings.HasPrefix(path, "/") &&
		!strings.HasSuffix(path, "/")
}

func (v *CodeValidator) validateTypeSpec(spec *ast.TypeSpec) error {
	// Check type name (should be PascalCase for exported types)
	if spec.Name.IsExported() {
		if !v.isPascalCase(spec.Name.Name) {
			return fmt.Errorf("exported type %s should be in PascalCase", spec.Name.Name)
		}
	}

	// Check struct fields if it's a struct type
	if structType, ok := spec.Type.(*ast.StructType); ok {
		return v.validateStructFields(spec.Name.Name, structType.Fields)
	}

	return nil
}

func (v *CodeValidator) validateStructFields(structName string, fields *ast.FieldList) error {
	var validationErrors []string

	for _, field := range fields.List {
		for _, name := range field.Names {
			// Check field name (should be PascalCase for exported fields)
			if name.IsExported() && !v.isPascalCase(name.Name) {
				validationErrors = append(validationErrors,
					fmt.Sprintf("exported field %s.%s should be in PascalCase",
						structName, name.Name))
			}
		}

		// Check tags if present
		if field.Tag != nil {
			if err := v.validateStructTag(field.Tag.Value); err != nil {
				validationErrors = append(validationErrors, err.Error())
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("struct validation failed:\n%s",
			strings.Join(validationErrors, "\n"))
	}

	return nil
}

func (v *CodeValidator) validateStructTag(tag string) error {
	// Remove surrounding quotes
	tag = strings.Trim(tag, "`")

	// Check common tag formats
	if !strings.Contains(tag, "json:\"") {
		return fmt.Errorf("missing json tag")
	}

	// Validate json tag format
	if !v.isValidJSONTag(tag) {
		return fmt.Errorf("invalid json tag format: %s", tag)
	}

	return nil
}

func (v *CodeValidator) isValidJSONTag(tag string) bool {
	// Basic json tag validation
	parts := strings.Split(tag, "json:\"")
	if len(parts) != 2 {
		return false
	}

	tagValue := strings.Split(parts[1], "\"")[0]
	tagParts := strings.Split(tagValue, ",")

	// Check if the field name part is valid
	if len(tagParts) > 0 {
		fieldName := tagParts[0]
		if fieldName != "-" && !v.isValidJSONFieldName(fieldName) {
			return false
		}
	}

	// Check tag options
	for i, opt := range tagParts {
		if i == 0 {
			continue
		}
		if opt != "omitempty" && opt != "string" {
			return false
		}
	}

	return true
}

func (v *CodeValidator) isValidJSONFieldName(name string) bool {
	// JSON field names should be lowercase with optional underscores
	for _, r := range name {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789_", r) {
			return false
		}
	}
	return true
}

func (v *CodeValidator) validateFuncDecl(decl *ast.FuncDecl) error {
	var validationErrors []string

	// Check function name
	if decl.Name.IsExported() {
		if !v.isPascalCase(decl.Name.Name) {
			validationErrors = append(validationErrors,
				fmt.Sprintf("exported function %s should be in PascalCase", decl.Name.Name))
		}
	}

	// Check receiver if it's a method
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		if len(recv.Names) > 0 {
			receiverName := recv.Names[0].Name
			if len(receiverName) > 3 {
				validationErrors = append(validationErrors,
					fmt.Sprintf("receiver name %s is too long, should be 1-3 characters", receiverName))
			}
		}
	}

	// Check parameters
	if err := v.validateFuncParams(decl.Type.Params); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Check return values
	if err := v.validateFuncResults(decl.Type.Results); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("function validation failed:\n%s",
			strings.Join(validationErrors, "\n"))
	}

	return nil
}

func (v *CodeValidator) validateFuncParams(fields *ast.FieldList) error {
	if fields == nil {
		return nil
	}

	var validationErrors []string

	for _, field := range fields.List {
		for _, name := range field.Names {
			if !v.isValidParamName(name.Name) {
				validationErrors = append(validationErrors,
					fmt.Sprintf("invalid parameter name: %s", name.Name))
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("parameter validation failed:\n%s",
			strings.Join(validationErrors, "\n"))
	}

	return nil
}

func (v *CodeValidator) validateFuncResults(fields *ast.FieldList) error {
	if fields == nil {
		return nil
	}

	var validationErrors []string

	for _, field := range fields.List {
		// Unnamed return values are fine
		if field.Names != nil {
			for _, name := range field.Names {
				if !v.isValidResultName(name.Name) {
					validationErrors = append(validationErrors,
						fmt.Sprintf("invalid result name: %s", name.Name))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("result validation failed:\n%s",
			strings.Join(validationErrors, "\n"))
	}

	return nil
}

func (v *CodeValidator) isPascalCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

func (v *CodeValidator) isValidParamName(name string) bool {
	// Parameter names should be short and descriptive
	return len(name) > 0 && len(name) <= 20 && v.isCamelCase(name)
}

func (v *CodeValidator) isValidResultName(name string) bool {
	// Result names should be short and descriptive
	return len(name) > 0 && len(name) <= 20 && v.isCamelCase(name)
}

func (v *CodeValidator) isCamelCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsLower(rune(name[0])) && !strings.Contains(name, "_")
}
