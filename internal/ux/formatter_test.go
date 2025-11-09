package ux

import (
	"bytes"
	"strings"
	"testing"
)

type testData struct {
	Name  string `json:"name" yaml:"name"`
	Value int    `json:"value" yaml:"value"`
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"json format", "json", false},
		{"yaml format", "yaml", false},
		{"text format", "text", false},
		{"empty format defaults to text", "", false},
		{"unknown format", "xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFormatter(tt.format, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFormatter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	formatter, err := NewFormatter("json", &FormatterOptions{Writer: &buf})
	if err != nil {
		t.Fatalf("NewFormatter() error = %v", err)
	}

	data := testData{Name: "test", Value: 42}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name": "test"`) {
		t.Errorf("JSON output missing expected field: %s", output)
	}
	if !strings.Contains(output, `"value": 42`) {
		t.Errorf("JSON output missing expected field: %s", output)
	}
}

func TestJSONFormatterCompact(t *testing.T) {
	var buf bytes.Buffer
	formatter, err := NewFormatter("json", &FormatterOptions{
		Writer:  &buf,
		Compact: true,
	})
	if err != nil {
		t.Fatalf("NewFormatter() error = %v", err)
	}

	data := testData{Name: "test", Value: 42}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	// Compact JSON should be single line (no indentation)
	if strings.Count(output, "\n") > 1 {
		t.Errorf("Compact JSON should be single line, got: %s", output)
	}
}

func TestYAMLFormatter(t *testing.T) {
	var buf bytes.Buffer
	formatter, err := NewFormatter("yaml", &FormatterOptions{Writer: &buf})
	if err != nil {
		t.Fatalf("NewFormatter() error = %v", err)
	}

	data := testData{Name: "test", Value: 42}
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name: test") {
		t.Errorf("YAML output missing expected field: %s", output)
	}
	if !strings.Contains(output, "value: 42") {
		t.Errorf("YAML output missing expected field: %s", output)
	}
}

func TestTextFormatter(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		want    string
		wantErr bool
	}{
		{
			name: "string data",
			data: "hello world",
			want: "hello world",
		},
		{
			name:    "complex type without String method",
			data:    testData{Name: "test", Value: 42},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter, err := NewFormatter("text", &FormatterOptions{Writer: &buf})
			if err != nil {
				t.Fatalf("NewFormatter() error = %v", err)
			}

			err = formatter.Format(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := strings.TrimSpace(buf.String())
				if output != tt.want {
					t.Errorf("Format() output = %q, want %q", output, tt.want)
				}
			}
		})
	}
}
