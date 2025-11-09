package ux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
