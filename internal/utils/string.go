package utils

import (
	"strings"
	"unicode"
)

func ToCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	for i := 0; i < len(words); i++ {
		words[i] = strings.Title(strings.ToLower(words[i]))
	}

	return strings.Join(words, "")
}

func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func ToLowerCamelCase(s string) string {
	s = ToCamelCase(s)
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func Pluralize(s string) string {
	// Very basic pluralization - you might want to use a proper library
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// StringContains checks if a string slice contains a specific string
func StringContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ToGoIdentifier converts a string to a valid Go identifier
func ToGoIdentifier(s string) string {
	// Split on non-alphanumeric characters
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	// Convert to camel case
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else {
			words[i] = strings.Title(strings.ToLower(word))
		}
	}

	return strings.Join(words, "")
}

// SanitizePackageName ensures the package name is valid Go package name
func SanitizePackageName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return '_'
	}, name)

	// Ensure it starts with a letter
	if !unicode.IsLetter(rune(name[0])) {
		name = "pkg_" + name
	}

	return name
}

// GetImportPath generates the import path for a package
func GetImportPath(baseImportPath, packageName string) string {
	return strings.TrimSuffix(baseImportPath, "/") + "/" + packageName
}
