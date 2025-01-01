package generator

import (
	"bytes"
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/utils"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

type TemplateEngine struct {
	config    *config.Config
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

func NewTemplateEngine(cfg *config.Config) (*TemplateEngine, error) {
	engine := &TemplateEngine{
		config:    cfg,
		templates: make(map[string]*template.Template),
	}

	// Initialize template function map
	engine.initFuncMap()

	// Load templates from directory
	if err := engine.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return engine, nil
}

func (e *TemplateEngine) initFuncMap() {
	e.funcMap = template.FuncMap{
		"toLower":      strings.ToLower,
		"toUpper":      strings.ToUpper,
		"toCamel":      utils.ToCamelCase,
		"toLowerCamel": utils.ToLowerCamelCase,
		"toSnake":      utils.ToSnakeCase,
		"trimPrefix":   strings.TrimPrefix,
		"trimSuffix":   strings.TrimSuffix,
		"join":         strings.Join,
		"hasPrefix":    strings.HasPrefix,
		"hasSuffix":    strings.HasSuffix,
		"contains":     strings.Contains,
		"replace":      strings.Replace,
		"quote":        strconv.Quote,
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
		"mul":          func(a, b int) int { return a * b },
		"div":          func(a, b int) int { return a / b },
	}
}

func (e *TemplateEngine) loadTemplates() error {
	templateDir := "templates"
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		templatePath := filepath.Join(templateDir, entry.Name())

		content, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		tmpl, err := template.New(name).Funcs(e.funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		e.templates[name] = tmpl
	}

	return nil
}

func (e *TemplateEngine) Execute(templateName string, data interface{}) ([]byte, error) {
	tmpl, ok := e.templates[templateName]
	if !ok {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	// Format Go code if it's a Go file
	if strings.HasSuffix(templateName, ".go") {
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("failed to format generated code: %w", err)
		}
		return formatted, nil
	}

	return buf.Bytes(), nil
}
