// Specular Slack Notifier Plugin
// Sends notifications to Slack channels via webhook.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	PluginName    = "slack-notifier"
	PluginVersion = "1.0.0"
)

// Request types
type NotifierRequest struct {
	Action string                 `json:"action"`
	Event  string                 `json:"event"`
	Data   map[string]interface{} `json:"data"`
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

// Slack message types
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Blocks      []SlackBlock `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type SlackBlock struct {
	Type   string      `json:"type"`
	Text   *BlockText  `json:"text,omitempty"`
	Fields []BlockText `json:"fields,omitempty"`
}

type BlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Attachment struct {
	Color  string `json:"color,omitempty"`
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	Footer string `json:"footer,omitempty"`
	Ts     int64  `json:"ts,omitempty"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Bytes()

	var request NotifierRequest
	if err := json.Unmarshal(input, &request); err != nil {
		respond(PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	var response PluginResponse
	switch request.Action {
	case "health":
		response = handleHealth()
	case "notify":
		response = handleNotify(request)
	default:
		response = PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", request.Action),
		}
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

func handleNotify(request NotifierRequest) PluginResponse {
	// Get webhook URL from config
	webhookURL, ok := request.Config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return PluginResponse{
			Success: false,
			Error:   "webhook_url is required in configuration",
		}
	}

	// Build Slack message based on event type
	message := buildSlackMessage(request.Event, request.Data)

	// Send to Slack
	if err := sendToSlack(webhookURL, message); err != nil {
		return PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to send to Slack: %v", err),
		}
	}

	return PluginResponse{
		Success: true,
		Result: map[string]interface{}{
			"message": fmt.Sprintf("Notification sent for event: %s", request.Event),
		},
	}
}

func buildSlackMessage(event string, data map[string]interface{}) SlackMessage {
	// Get common fields
	title := getString(data, "title", event)
	message := getString(data, "message", "")
	status := getString(data, "status", "info")

	// Color based on status
	color := "#36a64f" // green
	emoji := ":white_check_mark:"
	switch status {
	case "error", "failed":
		color = "#ff0000"
		emoji = ":x:"
	case "warning":
		color = "#ffcc00"
		emoji = ":warning:"
	case "info":
		color = "#36a64f"
		emoji = ":information_source:"
	case "success":
		color = "#36a64f"
		emoji = ":white_check_mark:"
	}

	// Build blocks for rich formatting
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &BlockText{
				Type: "plain_text",
				Text: fmt.Sprintf("%s %s", emoji, title),
			},
		},
	}

	if message != "" {
		blocks = append(blocks, SlackBlock{
			Type: "section",
			Text: &BlockText{
				Type: "mrkdwn",
				Text: message,
			},
		})
	}

	// Add fields for additional data
	fields := []BlockText{}
	for key, value := range data {
		if key == "title" || key == "message" || key == "status" {
			continue
		}
		fields = append(fields, BlockText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s:*\n%v", key, value),
		})
	}

	if len(fields) > 0 {
		blocks = append(blocks, SlackBlock{
			Type:   "section",
			Fields: fields,
		})
	}

	// Add context with timestamp
	blocks = append(blocks, SlackBlock{
		Type: "context",
		Fields: []BlockText{
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Specular* | %s", time.Now().Format(time.RFC3339)),
			},
		},
	})

	return SlackMessage{
		Blocks: blocks,
		Attachments: []Attachment{
			{
				Color: color,
			},
		},
	}
}

func sendToSlack(webhookURL string, message SlackMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("post to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func getString(data map[string]interface{}, key, defaultValue string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return defaultValue
}

func respond(response PluginResponse) {
	output, _ := json.Marshal(response)
	fmt.Println(string(output))
}
