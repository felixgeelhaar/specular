package ux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for yes/no confirmation
func Confirm(message string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	prompt := message
	if defaultYes {
		prompt += " (Y/n): "
	} else {
		prompt += " (y/N): "
	}

	fmt.Print(prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// PromptForPath prompts the user for a file path with a default value
func PromptForPath(message string, defaultPath string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [%s]: ", message, defaultPath)
	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultPath
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultPath
	}

	return response
}

// PromptForString prompts the user for a string value
func PromptForString(message string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	return response
}

// Select prompts the user to select from a list of options
func Select(message string, options []string, defaultIdx int) (string, int) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(message)
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = ">"
		}
		fmt.Printf(" %s %d. %s\n", marker, i+1, opt)
	}

	fmt.Printf("Enter selection [%d]: ", defaultIdx+1)
	response, err := reader.ReadString('\n')
	if err != nil {
		return options[defaultIdx], defaultIdx
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return options[defaultIdx], defaultIdx
	}

	var selection int
	_, err = fmt.Sscanf(response, "%d", &selection)
	if err != nil || selection < 1 || selection > len(options) {
		return options[defaultIdx], defaultIdx
	}

	return options[selection-1], selection - 1
}
