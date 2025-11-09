package ux

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Formatter defines the interface for output formatters.
// This enables consistent output formatting across all commands.
type Formatter interface {
	// Format writes the given data to the output writer
	Format(data interface{}) error
}

// FormatterOptions contains configuration for formatters
type FormatterOptions struct {
	// Writer is where output is written (defaults to os.Stdout)
	Writer io.Writer
	// NoColor disables colored output for text formatters
	NoColor bool
	// Compact enables compact output (no indentation for JSON/YAML)
	Compact bool
}

// NewFormatter creates a formatter based on the format string
func NewFormatter(format string, opts *FormatterOptions) (Formatter, error) {
	if opts == nil {
		opts = &FormatterOptions{Writer: os.Stdout}
	}
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	switch format {
	case "json":
		return &JSONFormatter{opts: opts}, nil
	case "yaml":
		return &YAMLFormatter{opts: opts}, nil
	case "text", "":
		return &TextFormatter{opts: opts}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (supported: text, json, yaml)", format)
	}
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	opts *FormatterOptions
}

// Format writes data as JSON
func (f *JSONFormatter) Format(data interface{}) error {
	encoder := json.NewEncoder(f.opts.Writer)
	if !f.opts.Compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// YAMLFormatter formats output as YAML
type YAMLFormatter struct {
	opts *FormatterOptions
}

// Format writes data as YAML
func (f *YAMLFormatter) Format(data interface{}) error {
	encoder := yaml.NewEncoder(f.opts.Writer)
	if !f.opts.Compact {
		encoder.SetIndent(2)
	}
	defer encoder.Close()
	return encoder.Encode(data)
}

// TextFormatter formats output as human-readable text
type TextFormatter struct {
	opts *FormatterOptions
}

// Format writes data as formatted text
// Note: TextFormatter requires data to implement a String() method
// or be a primitive type (string, int, bool, etc.)
func (f *TextFormatter) Format(data interface{}) error {
	switch v := data.(type) {
	case string:
		_, err := fmt.Fprintln(f.opts.Writer, v)
		return err
	case fmt.Stringer:
		_, err := fmt.Fprintln(f.opts.Writer, v.String())
		return err
	default:
		// For complex types, fall back to JSON with better error message
		return fmt.Errorf("text formatter requires data to implement String() method or be a primitive type")
	}
}

// Compile-time verification that formatters implement Formatter
var _ Formatter = (*JSONFormatter)(nil)
var _ Formatter = (*YAMLFormatter)(nil)
var _ Formatter = (*TextFormatter)(nil)
