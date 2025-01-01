package errors

import (
	"fmt"
)

type ErrorCode int

const (
	ErrCodeUnknown ErrorCode = iota
	ErrCodeInvalidInput
	ErrCodeParsingFailed
	ErrCodeValidationFailed
	ErrCodeTemplateError
	ErrCodeFileSystemError
	ErrCodeGenerationFailed
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// New creates a new error with the given code and message
func New(code ErrorCode, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap creates a new error wrapping an existing error
func Wrap(code ErrorCode, message string, err error) error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// InvalidInput creates an invalid input error
func InvalidInput(message string) error {
	return New(ErrCodeInvalidInput, message)
}

// ParsingFailed creates a parsing failed error
func ParsingFailed(err error) error {
	return Wrap(ErrCodeParsingFailed, "parsing failed", err)
}

// ValidationFailed creates a validation failed error
func ValidationFailed(err error) error {
	return Wrap(ErrCodeValidationFailed, "validation failed", err)
}

// TemplateError creates a template processing error
func TemplateError(err error) error {
	return Wrap(ErrCodeTemplateError, "template processing failed", err)
}

// FileSystemError creates a file system operation error
func FileSystemError(err error) error {
	return Wrap(ErrCodeFileSystemError, "file system operation failed", err)
}

// GenerationFailed creates a code generation error
func GenerationFailed(err error) error {
	return Wrap(ErrCodeGenerationFailed, "code generation failed", err)
}

// Is checks if the error is of a specific error code
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}
