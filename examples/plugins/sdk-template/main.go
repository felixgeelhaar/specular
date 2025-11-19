// Specular Plugin SDK Template
// This is a starting point for creating a Specular plugin in Go.
// Replace the TODO comments with your implementation.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Request structures (match Specular plugin protocol)

type PluginRequest struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type PluginResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

// Plugin metadata - TODO: Update these values
const (
	PluginName    = "my-plugin"
	PluginVersion = "1.0.0"
)

func main() {
	// Read request from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Bytes()

	var request PluginRequest
	if err := json.Unmarshal(input, &request); err != nil {
		respond(PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Handle actions
	var response PluginResponse
	switch request.Action {
	case "health":
		response = handleHealth()
	default:
		response = handleAction(request)
	}

	respond(response)
}

func handleHealth() PluginResponse {
	return PluginResponse{
		Success: true,
		Result: HealthResponse{
			Status:  "healthy",
			Version: PluginVersion,
			Name:    PluginName,
		},
	}
}

func handleAction(request PluginRequest) PluginResponse {
	// TODO: Implement your plugin logic here
	// Access configuration with: request.Config["key"]
	// Access parameters with: request.Params["key"]

	// Example: Echo back the action
	result := map[string]interface{}{
		"message": fmt.Sprintf("Received action: %s", request.Action),
		"params":  request.Params,
	}

	return PluginResponse{
		Success: true,
		Result:  result,
	}
}

func respond(response PluginResponse) {
	output, _ := json.Marshal(response)
	fmt.Println(string(output))
}

// Example implementations for different plugin types:

// For Notifier plugins:
// type NotifierRequest struct {
//     Action string                 `json:"action"`
//     Event  string                 `json:"event"`
//     Data   map[string]interface{} `json:"data"`
//     Config map[string]interface{} `json:"config,omitempty"`
// }

// For Validator plugins:
// type ValidatorRequest struct {
//     Action  string                 `json:"action"`
//     Content string                 `json:"content"`
//     Rules   map[string]interface{} `json:"rules,omitempty"`
//     Config  map[string]interface{} `json:"config,omitempty"`
// }
// type ValidatorResponse struct {
//     Valid    bool             `json:"valid"`
//     Messages []ValidatorIssue `json:"messages,omitempty"`
// }

// For Formatter plugins:
// type FormatterRequest struct {
//     Action string                 `json:"action"`
//     Data   interface{}            `json:"data"`
//     Format string                 `json:"format"`
//     Config map[string]interface{} `json:"config,omitempty"`
// }
// type FormatterResponse struct {
//     Output string `json:"output"`
// }
