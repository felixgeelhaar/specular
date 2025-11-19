package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var platformCmd = &cobra.Command{
	Use:   "platform",
	Short: "Manage platform resources (PRO feature)",
	Long: `Manage resources on the Specular platform.

Platform commands require authentication. Use 'specular auth login' first.

Subcommands:
  session   Manage AI sessions
  project   Manage projects

Examples:
  specular platform session list --project-id <id>
  specular platform session create --project-id <id> --title "My Session"
  specular platform project list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var platformSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage AI sessions",
	Long: `Manage AI sessions on the Specular platform.

Sessions track conversations with AI providers, including message history,
token usage, and costs.

Subcommands:
  list      List all sessions for a project
  create    Create a new session
  show      Show session details
  messages  List messages in a session
  send      Send a message to a session

Examples:
  specular platform session list --project-id <id>
  specular platform session create --project-id <id> --title "My Session"
  specular platform session show <session-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var platformSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project-id")
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		if projectID == "" {
			return fmt.Errorf("--project-id is required")
		}

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		sessions, err := client.ListSessions(projectID, page, pageSize)
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions.Sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		fmt.Printf("Sessions (page %d):\n\n", sessions.Page)
		for _, s := range sessions.Sessions {
			fmt.Printf("ID:       %s\n", s.ID)
			fmt.Printf("Title:    %s\n", s.Title)
			fmt.Printf("Status:   %s\n", s.Status)
			fmt.Printf("Provider: %s/%s\n", s.Provider, s.Model)
			fmt.Printf("Messages: %d\n", s.MessageCount)
			fmt.Printf("Tokens:   %d\n", s.TokensUsed)
			fmt.Printf("Cost:     $%.4f\n", s.EstimatedCost)
			fmt.Printf("Created:  %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Println("---")
		}

		return nil
	},
}

var platformSessionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new session",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project-id")
		title, _ := cmd.Flags().GetString("title")
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")
		tagsStr, _ := cmd.Flags().GetString("tags")

		if projectID == "" {
			return fmt.Errorf("--project-id is required")
		}
		if title == "" {
			return fmt.Errorf("--title is required")
		}

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		session, err := client.CreateSession(projectID, title, provider, model, nil, tags)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		fmt.Println("Session created!")
		fmt.Printf("ID:       %s\n", session.ID)
		fmt.Printf("Title:    %s\n", session.Title)
		fmt.Printf("Provider: %s/%s\n", session.Provider, session.Model)

		return nil
	},
}

var platformSessionShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		session, err := client.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}

		fmt.Printf("ID:         %s\n", session.ID)
		fmt.Printf("Title:      %s\n", session.Title)
		fmt.Printf("Status:     %s\n", session.Status)
		fmt.Printf("Provider:   %s\n", session.Provider)
		fmt.Printf("Model:      %s\n", session.Model)
		fmt.Printf("Messages:   %d\n", session.MessageCount)
		fmt.Printf("Tokens:     %d\n", session.TokensUsed)
		fmt.Printf("Est. Cost:  $%.4f\n", session.EstimatedCost)
		if len(session.Tags) > 0 {
			fmt.Printf("Tags:       %s\n", strings.Join(session.Tags, ", "))
		}
		fmt.Printf("Created:    %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:    %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var platformSessionMessagesCmd = &cobra.Command{
	Use:   "messages <session-id>",
	Short: "List messages in a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		messages, err := client.ListMessages(sessionID, page, pageSize)
		if err != nil {
			return fmt.Errorf("failed to list messages: %w", err)
		}

		if len(messages.Messages) == 0 {
			fmt.Println("No messages found.")
			return nil
		}

		for _, m := range messages.Messages {
			roleLabel := "User"
			if m.Role == "assistant" {
				roleLabel = "Assistant"
			} else if m.Role == "system" {
				roleLabel = "System"
			}

			fmt.Printf("[%s] %s\n", roleLabel, m.CreatedAt.Format("15:04:05"))
			content := m.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			fmt.Printf("%s\n", content)
			if m.TotalTokens > 0 {
				fmt.Printf("(tokens: %d)\n", m.TotalTokens)
			}
			fmt.Println("---")
		}

		return nil
	},
}

var platformSessionSendCmd = &cobra.Command{
	Use:   "send <session-id> <message>",
	Short: "Send a message to a session",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		message := strings.Join(args[1:], " ")

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		fmt.Println("Sending message...")

		response, err := client.SendMessage(sessionID, message)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		fmt.Println("\nAssistant:")
		fmt.Println(response.AssistantMessage.Content)
		fmt.Printf("\n(tokens: %d, cost: $%.4f)\n",
			response.AssistantMessage.TotalTokens,
			response.Session.EstimatedCost)

		return nil
	},
}

func init() {
	// Session subcommands
	platformSessionCmd.AddCommand(platformSessionListCmd)
	platformSessionCmd.AddCommand(platformSessionCreateCmd)
	platformSessionCmd.AddCommand(platformSessionShowCmd)
	platformSessionCmd.AddCommand(platformSessionMessagesCmd)
	platformSessionCmd.AddCommand(platformSessionSendCmd)

	// Flags for list command
	platformSessionListCmd.Flags().String("project-id", "", "Project ID (required)")
	platformSessionListCmd.Flags().Int("page", 1, "Page number")
	platformSessionListCmd.Flags().Int("page-size", 10, "Items per page")

	// Flags for create command
	platformSessionCreateCmd.Flags().String("project-id", "", "Project ID (required)")
	platformSessionCreateCmd.Flags().String("title", "", "Session title (required)")
	platformSessionCreateCmd.Flags().String("provider", "openai", "AI provider")
	platformSessionCreateCmd.Flags().String("model", "gpt-4", "AI model")
	platformSessionCreateCmd.Flags().String("tags", "", "Comma-separated tags")

	// Flags for messages command
	platformSessionMessagesCmd.Flags().Int("page", 1, "Page number")
	platformSessionMessagesCmd.Flags().Int("page-size", 20, "Items per page")

	// Add session to platform
	platformCmd.AddCommand(platformSessionCmd)

	// Add platform to root
	rootCmd.AddCommand(platformCmd)
}
