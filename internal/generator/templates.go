package generator

import (
	"bytes"
	"fmt"
	"github.com/chashtager/opensdkraft/internal/config"
	"github.com/chashtager/opensdkraft/internal/errors"
	"github.com/chashtager/opensdkraft/internal/logging"
	"github.com/chashtager/opensdkraft/internal/utils"
	"go/format"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

type TemplateEngine struct {
	config          *config.Config
	templates       map[string]*template.Template
	cache           *templateCache
	funcMap         template.FuncMap
	customFunctions map[string]interface{}
	logger          *logging.Logger
}

func NewTemplateEngine(cfg *config.Config, logger *logging.Logger) (*TemplateEngine, error) {
	engine := &TemplateEngine{
		config:          cfg,
		templates:       make(map[string]*template.Template),
		cache:           newTemplateCache(),
		customFunctions: make(map[string]interface{}),
		logger:          logger,
	}

	// Initialize template function map
	engine.initFuncMap()

	// Load templates from directory
	if err := engine.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return engine, nil
}

type templateCache struct {
	cache map[uint64][]byte
	mu    sync.RWMutex
}

func newTemplateCache() *templateCache {
	return &templateCache{
		cache: make(map[uint64][]byte),
	}
}

func (c *templateCache) get(key uint64) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[key]
}

func (c *templateCache) set(key uint64, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = value
}

func (e *TemplateEngine) AddCustomFunc(name string, fn interface{}) error {
	if _, exists := e.customFunctions[name]; exists {
		return errors.InvalidInput(fmt.Sprintf("function %s already exists", name))
	}

	e.customFunctions[name] = fn
	e.initFuncMap()
	return nil
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

	// Ensure template directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory does not exist: %s", templateDir)
	}

	// Keep track of loaded templates
	loadedTemplates := make(map[string]bool)

	// Walk through the template directory recursively
	err := filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access %s: %w", path, err)
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process files with .tmpl extension
		if !strings.HasSuffix(d.Name(), ".tmpl") {
			return nil
		}

		// Extract template name relative to the root templateDir
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return fmt.Errorf("failed to determine relative path for %s: %w", path, err)
		}

		// Use the filename (without extension) as the template name
		name := strings.TrimSuffix(relPath, ".tmpl")

		if loadedTemplates[name] {
			return fmt.Errorf("duplicate template name found: %s", name)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		tmpl, err := template.New(name).Funcs(e.funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		e.logger.Debug("Loaded template: %s from %s", name, path)

		e.templates[name] = tmpl
		loadedTemplates[name] = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Check if we loaded at least the minimum required templates
	requiredTemplates := []string{"model", "client", "operation"}
	var missingTemplates []string
	for _, required := range requiredTemplates {
		normalized := filepath.ToSlash(required)
		found := false
		for loaded := range loadedTemplates {
			if filepath.ToSlash(loaded) == normalized {
				found = true
				break
			}
		}
		if !found {
			missingTemplates = append(missingTemplates, required)
		}
	}

	e.logger.Info("Successfully loaded %d templates", len(loadedTemplates))

	return nil
}

//func (e *TemplateEngine) loadTemplates() error {
//	templates := []struct {
//		name     string
//		path     string
//		required bool
//	}{
//		{"model", "templates/model.tmpl", true},
//		{"client", "templates/client.tmpl", true},
//		{"operation", "templates/operation.tmpl", true},
//		{"model_test", "templates/tests/model_test.tmpl", true},
//		{"client_test", "templates/tests/client_test.tmpl", true},
//		{"operation_test", "templates/tests/operation_test.tmpl", true},
//		{"test_helpers", "templates/tests/test_helpers.tmpl", true},
//	}
//
//	for _, t := range templates {
//		content, err := os.ReadFile(t.path)
//		if err != nil {
//			if t.required {
//				return fmt.Errorf("failed to read required template %s: %w", t.name, err)
//			}
//			continue
//		}
//
//		tmpl, err := template.New(t.name).Funcs(e.funcMap).Parse(string(content))
//		if err != nil {
//			return fmt.Errorf("failed to parse template %s: %w", t.name, err)
//		}
//
//		e.templates[t.name] = tmpl
//	}
//
//	return nil
//}

func (e *TemplateEngine) Execute(templateName string, data interface{}) ([]byte, error) {
	//Generate cache key
	h := fnv.New64()
	h.Write([]byte(fmt.Sprintf("%s-%v", templateName, data)))
	key := h.Sum64()

	// Check cache
	if cached := e.cache.get(key); cached != nil {
		return cached, nil
	}

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

	result := buf.Bytes()
	e.cache.set(key, result)

	return result, nil
}

func (e *TemplateEngine) ValidateTemplate(content string) error {
	_, err := template.New("validator").Funcs(e.funcMap).Parse(content)
	if err != nil {
		return errors.TemplateError(err)
	}
	return nil
}
