package log

import (
	"io"
	"os"
)

// Format represents the output format for logs
type Format int

const (
	// FormatJSON outputs logs in JSON format
	FormatJSON Format = iota
	// FormatText outputs logs in human-readable text format
	FormatText
)

// String returns the string representation of the format
func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatText:
		return "text"
	default:
		return "json"
	}
}

// ParseFormat parses a string into a Format
func ParseFormat(s string) Format {
	switch s {
	case "json", "JSON":
		return FormatJSON
	case "text", "TEXT", "console":
		return FormatText
	default:
		return FormatJSON
	}
}

// Output represents where logs should be written
type Output struct {
	writer io.Writer
}

// Writer returns the underlying io.Writer
func (o Output) Writer() io.Writer {
	return o.writer
}

// NewOutput creates an Output from an io.Writer
func NewOutput(w io.Writer) Output {
	return Output{writer: w}
}

// OutputStdout creates an Output that writes to stdout
func OutputStdout() Output {
	return Output{writer: os.Stdout}
}

// OutputStderr creates an Output that writes to stderr
func OutputStderr() Output {
	return Output{writer: os.Stderr}
}

// Config holds configuration for the logger
type Config struct {
	// Level is the minimum log level to output
	Level Level

	// Format is the output format (JSON or Text)
	Format Format

	// Output is where logs should be written
	Output Output

	// AddSource includes source file and line number in logs
	AddSource bool

	// ServiceName is the name of the service (for tracing correlation)
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string
}

// DefaultConfig returns a sensible default configuration
// Logs at INFO level in JSON format to stdout
func DefaultConfig() Config {
	return Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         OutputStdout(),
		AddSource:      false,
		ServiceName:    "specular",
		ServiceVersion: "dev",
	}
}

// DevelopmentConfig returns a configuration suitable for development
// Logs at DEBUG level in text format to stdout with source location
func DevelopmentConfig() Config {
	return Config{
		Level:          LevelDebug,
		Format:         FormatText,
		Output:         OutputStdout(),
		AddSource:      true,
		ServiceName:    "specular",
		ServiceVersion: "dev",
	}
}

// ProductionConfig returns a configuration suitable for production
// Logs at INFO level in JSON format to stdout
func ProductionConfig() Config {
	return Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         OutputStdout(),
		AddSource:      false,
		ServiceName:    "specular",
		ServiceVersion: "unknown",
	}
}
