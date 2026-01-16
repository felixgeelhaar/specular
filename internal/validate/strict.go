// Package validate provides input validation utilities for CLI commands.
package validate

import (
	"fmt"
	"io"
	"strings"
)

// StrictMode controls whether warnings should be treated as errors
type StrictMode struct {
	// Enabled treats warnings as errors
	Enabled bool
	// FailOnWarnings explicitly fails when warnings are present
	FailOnWarnings bool
}

// DefaultStrictMode returns the default strict mode settings
func DefaultStrictMode() StrictMode {
	return StrictMode{
		Enabled:        false,
		FailOnWarnings: false,
	}
}

// StrictValidator wraps a ValidationResult and provides strict mode checking
type StrictValidator struct {
	Result *ValidationResult
	Mode   StrictMode
}

// NewStrictValidator creates a new strict validator with the given mode
func NewStrictValidator(mode StrictMode) *StrictValidator {
	return &StrictValidator{
		Result: NewValidationResult(),
		Mode:   mode,
	}
}

// AddError adds an error-level issue
func (v *StrictValidator) AddError(field, value, message string) {
	v.Result.AddError(field, value, message)
}

// AddWarning adds a warning-level issue (or error in strict mode)
func (v *StrictValidator) AddWarning(field, value, message string) {
	if v.Mode.Enabled {
		v.Result.AddError(field, value, message+" (strict mode)")
	} else {
		v.Result.AddWarning(field, value, message)
	}
}

// AddInfo adds an info-level issue
func (v *StrictValidator) AddInfo(field, value, message string) {
	v.Result.AddInfo(field, value, message)
}

// ShouldFail returns true if execution should be stopped
func (v *StrictValidator) ShouldFail() bool {
	if v.Result.HasErrors() {
		return true
	}
	if v.Mode.FailOnWarnings && v.Result.HasWarnings() {
		return true
	}
	return false
}

// GetError returns an error if validation failed, nil otherwise
func (v *StrictValidator) GetError() error {
	if v.Result.HasErrors() {
		return v.Result.Error()
	}
	if v.Mode.FailOnWarnings && v.Result.HasWarnings() {
		warnings := v.Result.Warnings()
		if len(warnings) == 1 {
			return fmt.Errorf("validation warning (strict mode): %s", warnings[0].Message)
		}
		var messages []string
		for _, w := range warnings {
			messages = append(messages, w.Message)
		}
		return fmt.Errorf("validation failed with %d warnings (strict mode): %s",
			len(warnings), strings.Join(messages, "; "))
	}
	return nil
}

// PrintReport prints a formatted validation report
func (v *StrictValidator) PrintReport(w io.Writer, useColor bool) {
	PrintValidationResult(w, v.Result, useColor)
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// PrintValidationResult prints a formatted validation report to the writer
func PrintValidationResult(w io.Writer, result *ValidationResult, useColor bool) {
	if result == nil || !result.HasIssues() {
		return
	}

	errors, warnings, infos := result.Count()

	// Print header
	if useColor {
		_, _ = fmt.Fprintf(w, "\n%s%sValidation Report%s\n", colorBold, colorCyan, colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "\nValidation Report\n")
	}
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 50))

	// Print errors
	if errors > 0 {
		if useColor {
			_, _ = fmt.Fprintf(w, "%s%sErrors (%d):%s\n", colorBold, colorRed, errors, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "Errors (%d):\n", errors)
		}
		for _, issue := range result.Errors() {
			printIssue(w, issue, useColor)
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print warnings
	if warnings > 0 {
		if useColor {
			_, _ = fmt.Fprintf(w, "%s%sWarnings (%d):%s\n", colorBold, colorYellow, warnings, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "Warnings (%d):\n", warnings)
		}
		for _, issue := range result.Warnings() {
			printIssue(w, issue, useColor)
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print info messages
	if infos > 0 {
		if useColor {
			_, _ = fmt.Fprintf(w, "%s%sInfo (%d):%s\n", colorBold, colorCyan, infos, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "Info (%d):\n", infos)
		}
		for _, issue := range result.Issues {
			if issue.Severity == SeverityInfo {
				printIssue(w, issue, useColor)
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print summary
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 50))
	summaryParts := make([]string, 0, 3)
	if errors > 0 {
		if useColor {
			summaryParts = append(summaryParts, fmt.Sprintf("%s%d error(s)%s", colorRed, errors, colorReset))
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("%d error(s)", errors))
		}
	}
	if warnings > 0 {
		if useColor {
			summaryParts = append(summaryParts, fmt.Sprintf("%s%d warning(s)%s", colorYellow, warnings, colorReset))
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("%d warning(s)", warnings))
		}
	}
	if infos > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d info", infos))
	}
	_, _ = fmt.Fprintf(w, "Summary: %s\n", strings.Join(summaryParts, ", "))
}

// printIssue prints a single validation issue
func printIssue(w io.Writer, issue ValidationIssue, useColor bool) {
	symbol := issue.Severity.Symbol()
	prefix := ""

	if useColor {
		switch issue.Severity {
		case SeverityError:
			prefix = colorRed
		case SeverityWarning:
			prefix = colorYellow
		case SeverityInfo:
			prefix = colorCyan
		}
	}

	// Format: ✗ [code] field: message (got: "value")
	var parts []string
	parts = append(parts, symbol)

	if issue.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", issue.Code))
	}

	if issue.Field != "" {
		parts = append(parts, fmt.Sprintf("%s:", issue.Field))
	}

	parts = append(parts, issue.Message)

	if issue.Value != "" {
		displayValue := issue.Value
		if len(displayValue) > 30 {
			displayValue = displayValue[:27] + "..."
		}
		parts = append(parts, fmt.Sprintf("(got: %q)", displayValue))
	}

	if useColor {
		_, _ = fmt.Fprintf(w, "  %s%s%s\n", prefix, strings.Join(parts, " "), colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "  %s\n", strings.Join(parts, " "))
	}
}

// PrintWarnings prints only warning-level issues (for non-strict mode)
func PrintWarnings(w io.Writer, result *ValidationResult, useColor bool) {
	if result == nil || !result.HasWarnings() {
		return
	}

	warnings := result.Warnings()

	if useColor {
		_, _ = fmt.Fprintf(w, "\n%s%s⚠ Warnings (%d):%s\n", colorBold, colorYellow, len(warnings), colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "\n⚠ Warnings (%d):\n", len(warnings))
	}

	for _, issue := range warnings {
		printIssue(w, issue, useColor)
	}
}

// ValidateWithResult validates a value and adds the result to a ValidationResult
func ValidateWithResult(result *ValidationResult, severity Severity, field, value string, validateFn func(string) error) {
	if err := validateFn(value); err != nil {
		result.AddIssue(severity, field, value, err.Error(), "")
	}
}

// WarnIfEmpty adds a warning if the value is empty
func WarnIfEmpty(result *ValidationResult, field, value string) {
	if strings.TrimSpace(value) == "" {
		result.AddWarning(field, "", "value is empty")
	}
}

// WarnIfDeprecated adds a warning about deprecated usage
func WarnIfDeprecated(result *ValidationResult, field, deprecatedValue, replacement string) {
	result.AddWarning(field, deprecatedValue,
		fmt.Sprintf("deprecated, use %q instead", replacement))
}

// WarnIfNotRecommended adds a warning about non-recommended values
func WarnIfNotRecommended(result *ValidationResult, field, value, recommendation string) {
	result.AddWarning(field, value,
		fmt.Sprintf("not recommended: %s", recommendation))
}
