package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/interview"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var interviewCmd = &cobra.Command{
	Use:   "interview",
	Short: "Interactive interview mode to generate spec from Q&A",
	Long: `Launch an interactive interview session that guides you through
creating a best-practice specification from natural language inputs.

Supports presets (web-app, api-service, cli-tool, microservice, data-pipeline)
and strict mode for enhanced validation.`,
	RunE: runInterview,
}

func runInterview(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	out := cmd.Flags().Lookup("out").Value.String()
	preset := cmd.Flags().Lookup("preset").Value.String()
	strict := cmd.Flags().Lookup("strict").Value.String() == "true"
	tui := cmd.Flags().Lookup("tui").Value.String() == "true"
	list := cmd.Flags().Lookup("list").Value.String() == "true"

	// Use smart default for output if not changed
	if !cmd.Flags().Changed("out") {
		out = defaults.SpecFile()
	}

	// List available presets
	if list {
		return listPresets()
	}

	// Interactive preset selection if not provided
	if preset == "" && !cmd.Flags().Changed("preset") {
		fmt.Println("Select a preset for your project:")
		presets := []string{
			"web-app - Web application with UI and backend",
			"api-service - RESTful API service",
			"cli-tool - Command-line interface tool",
			"microservice - Microservice component",
			"data-pipeline - Data processing pipeline",
		}
		selected, _ := ux.Select("Choose preset:", presets, 0)
		// Extract just the preset name (before the dash)
		preset = strings.Split(selected, " ")[0]
	} else if preset == "" {
		return ux.NewErrorWithSuggestion(
			fmt.Errorf("preset is required"),
			"Use --list to see available presets or run without --preset for interactive selection",
		)
	}

	// Run TUI or CLI interview
	if tui {
		return runTUIInterview(preset, strict, out)
	}

	// Run CLI interview
	return runCLIInterview(preset, strict, out)
}

func listPresets() error {
	fmt.Println("Available interview presets:")

	presets := interview.ListPresets()
	for _, p := range presets {
		fmt.Printf("  %s\n", p.Name)
		fmt.Printf("    %s\n", p.Description)
		fmt.Printf("    Questions: %d\n\n", len(p.Questions))
	}

	return nil
}

//nolint:gocyclo // Interview flow complexity is acceptable for user interaction
func runCLIInterview(preset string, strict bool, out string) error {
	// Create interview engine
	engine, err := interview.NewEngine(preset, strict)
	if err != nil {
		return ux.FormatError(err, "creating interview engine")
	}

	fmt.Printf("=== AI-Dev Interview Mode ===\n")
	fmt.Printf("Preset: %s\n", preset)
	fmt.Printf("Strict mode: %v\n\n", strict)

	// Start interview
	if startErr := engine.Start(); startErr != nil {
		return ux.FormatError(startErr, "starting interview")
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Interview loop
	for !engine.IsComplete() {
		q, qErr := engine.CurrentQuestion()
		if qErr != nil {
			return ux.FormatError(qErr, "getting current question")
		}

		if q == nil {
			break // Interview complete
		}

		// Display question
		fmt.Printf("[%d%%] %s\n", int(engine.Progress()), q.Text)
		if q.Description != "" {
			fmt.Printf("     %s\n", q.Description)
		}

		// Show choices for choice/yesno questions
		switch q.Type {
		case interview.QuestionTypeYesNo:
			fmt.Printf("     Options: yes, no\n")
		case interview.QuestionTypeChoice:
			fmt.Printf("     Options:\n")
			for i, choice := range q.Choices {
				fmt.Printf("       %d. %s\n", i+1, choice)
			}
		case interview.QuestionTypeMulti:
			fmt.Printf("     (Enter each item on a new line, empty line to finish)\n")
		}

		// Mark required questions
		if q.Required {
			fmt.Printf("     (required)\n")
		}

		fmt.Printf("\n> ")

		// Read answer
		var answer interview.Answer

		if q.Type == interview.QuestionTypeMulti {
			// Read multi-line input
			values := []string{}
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					break
				}
				values = append(values, strings.TrimSpace(line))
			}
			answer.Values = values
		} else {
			// Read single line
			if !scanner.Scan() {
				return fmt.Errorf("failed to read input")
			}
			answer.Value = strings.TrimSpace(scanner.Text())

			// Convert choice number to text
			if q.Type == interview.QuestionTypeChoice {
				answer.Value = normalizeChoice(answer.Value, q.Choices)
			}
		}

		// Submit answer
		_, err = engine.Answer(answer)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			if strict {
				return err
			}
			fmt.Println("Please try again.")
			continue
		}

		fmt.Println()
	}

	// Generate spec from answers
	fmt.Println("\nGenerating specification from your answers...")

	result, err := engine.GetResult()
	if err != nil {
		return ux.FormatError(err, "generating spec from answers")
	}

	// Save spec
	if saveErr := spec.SaveSpec(result.Spec, out); saveErr != nil {
		return ux.FormatError(saveErr, "saving spec file")
	}

	fmt.Printf("\n✓ Specification generated successfully!\n")
	fmt.Printf("  Output: %s\n", out)
	fmt.Printf("  Product: %s\n", result.Spec.Product)
	fmt.Printf("  Features: %d\n", len(result.Spec.Features))
	fmt.Printf("  Generation time: %dms\n", result.Duration)

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review and edit: %s\n", out)
	fmt.Printf("  2. Validate spec: ai-dev spec validate --in %s\n", out)
	fmt.Printf("  3. Generate lock: ai-dev spec lock --in %s --out .specular/spec.lock.json\n", out)
	fmt.Printf("  4. Create plan: ai-dev plan --in %s --lock .specular/spec.lock.json --out plan.json\n", out)

	return nil
}

func runTUIInterview(preset string, strict bool, out string) error {
	// Create interview engine
	engine, err := interview.NewEngine(preset, strict)
	if err != nil {
		return ux.FormatError(err, "creating interview engine")
	}

	fmt.Printf("=== Specular Interview Mode (TUI) ===\n")
	fmt.Printf("Preset: %s\n", preset)
	fmt.Printf("Strict mode: %v\n\n", strict)
	fmt.Println("Starting interactive interview...")
	fmt.Println()

	// Run TUI interview
	result, err := tui.RunInterview(engine)
	if err != nil {
		return ux.FormatError(err, "running TUI interview")
	}

	// Save result
	err = tui.SaveResult(result, out)
	if err != nil {
		return ux.FormatError(err, "saving interview result")
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review and edit: %s\n", out)
	fmt.Printf("  2. Validate spec: specular spec validate --in %s\n", out)
	fmt.Printf("  3. Generate lock: specular spec lock --in %s --out .specular/spec.lock.json\n", out)
	fmt.Printf("  4. Create plan: specular plan --in %s --lock .specular/spec.lock.json --out plan.json\n", out)

	return nil
}

// normalizeChoice converts a choice number or partial match to full choice text
func normalizeChoice(input string, choices []string) string {
	// Try to parse as number
	for i, choice := range choices {
		if input == fmt.Sprintf("%d", i+1) {
			return choice
		}
	}

	// Try case-insensitive match
	for _, choice := range choices {
		if strings.EqualFold(input, choice) {
			return choice
		}
	}

	// Return as-is if no match
	return input
}

func init() {
	rootCmd.AddCommand(interviewCmd)

	interviewCmd.Flags().StringP("out", "o", ".specular/spec.yaml", "Output path for generated spec")
	interviewCmd.Flags().String("preset", "", "Use a preset template (use --list to see options)")
	interviewCmd.Flags().Bool("strict", false, "Enable strict validation mode")
	interviewCmd.Flags().Bool("tui", false, "Use interactive terminal UI mode")
	interviewCmd.Flags().Bool("list", false, "List available presets")
}
