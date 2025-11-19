package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/felixgeelhaar/specular/internal/platform"
)

func main() {
	// Create a new platform client
	client := platform.NewClient("http://localhost:8000")

	fmt.Println("=== Specular Platform Integration Demo ===\n")

	// Step 1: Register or login
	fmt.Println("1. Authenticating user...")
	var loginResp *platform.LoginResponse
	loginResp, err := client.Register(
		"demouser",
		"demo@specular.dev",
		"SecurePassword123!",
		"Demo",
		"User",
	)
	if err != nil {
		// If user already exists, try to login
		fmt.Println("  User already exists, logging in...")
		loginResp, err = client.Login("demo@specular.dev", "SecurePassword123!")
		if err != nil {
			log.Fatalf("Failed to login: %v", err)
		}
	} else {
		fmt.Printf("✓ Registered user: %s %s (%s)\n", loginResp.User.FirstName, loginResp.User.LastName, loginResp.User.Email)
	}
	fmt.Printf("✓ Access token: %s...\n\n", loginResp.AccessToken[:20])

	// Step 2: Get current user
	fmt.Println("2. Getting current user information...")
	user, err := client.GetCurrentUser()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}
	fmt.Printf("✓ Current user: %s %s (ID: %s)\n\n", user.FirstName, user.LastName, user.ID)

	// Step 3: Create a project (or use existing one)
	fmt.Println("3. Creating a new project...")
	project, err := client.CreateProject(
		"AI Development Platform",
		"Building an AI-powered development assistant",
		"PRIVATE",
		map[string]interface{}{
			"tech_stack": []string{"Go", "React", "PostgreSQL"},
			"team_size":  1,
		},
	)
	if err != nil {
		fmt.Println("  Project already exists, fetching existing projects...")
		// If project exists, get the first project from the list
		projects, listErr := client.ListProjects(1, 10)
		if listErr != nil {
			log.Fatalf("Failed to list projects: %v", listErr)
		}
		if len(projects.Projects) == 0 {
			log.Fatalf("No projects found and failed to create new one: %v", err)
		}
		project = &projects.Projects[0]
		fmt.Printf("✓ Using existing project: %s (ID: %s)\n", project.Name, project.ID)
	} else {
		fmt.Printf("✓ Created project: %s (ID: %s)\n", project.Name, project.ID)
	}
	fmt.Printf("  Description: %s\n\n", project.Description)

	// Step 4: List projects
	fmt.Println("4. Listing all projects...")
	projects, err := client.ListProjects(1, 10)
	if err != nil {
		log.Fatalf("Failed to list projects: %v", err)
	}
	fmt.Printf("✓ Found %d project(s)\n", projects.TotalCount)
	for _, p := range projects.Projects {
		fmt.Printf("  - %s: %s\n", p.Name, p.Description)
	}
	fmt.Println()

	// Step 5: Create an AI session
	fmt.Println("5. Creating an AI session...")
	session, err := client.CreateSession(
		project.ID,
		"Design Database Schema",
		"openai",
		"gpt-4",
		map[string]interface{}{
			"goal": "Design a scalable database schema for user management",
		},
		[]string{"database", "design", "architecture"},
	)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	fmt.Printf("✓ Created session: %s (ID: %s)\n", session.Title, session.ID)
	fmt.Printf("  Provider: %s, Model: %s\n", session.Provider, session.Model)
	fmt.Printf("  Status: %s\n\n", session.Status)

	// Step 6: Send a message to the AI
	fmt.Println("6. Sending message to AI...")
	messageResp, err := client.SendMessage(
		session.ID,
		"Can you help me design a user table with email, password hash, and profile fields?",
	)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	fmt.Printf("✓ User message sent (ID: %s)\n", messageResp.UserMessage.ID)
	fmt.Printf("✓ AI response received (ID: %s)\n", messageResp.AssistantMessage.ID)
	fmt.Printf("  Tokens used: %d\n", messageResp.AssistantMessage.TotalTokens)
	fmt.Printf("  Response preview: %s...\n\n", truncate(messageResp.AssistantMessage.Content, 100))

	// Step 7: List messages
	fmt.Println("7. Listing session messages...")
	messages, err := client.ListMessages(session.ID, 1, 10)
	if err != nil {
		log.Fatalf("Failed to list messages: %v", err)
	}
	fmt.Printf("✓ Found %d message(s)\n", messages.TotalCount)
	for _, m := range messages.Messages {
		fmt.Printf("  [%s] %s (tokens: %d)\n", m.Role, truncate(m.Content, 60), m.TotalTokens)
	}
	fmt.Println()

	// Step 8: Get session statistics
	fmt.Println("8. Getting updated session statistics...")
	updatedSession, err := client.GetSession(session.ID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}
	fmt.Printf("✓ Session statistics:\n")
	fmt.Printf("  Messages: %d\n", updatedSession.MessageCount)
	fmt.Printf("  Tokens used: %d\n", updatedSession.TokensUsed)
	fmt.Printf("  Estimated cost: $%.4f\n\n", updatedSession.EstimatedCost)

	// Step 9: Complete the session
	fmt.Println("9. Marking session as completed...")
	completedSession, err := client.CompleteSession(session.ID)
	if err != nil {
		log.Fatalf("Failed to complete session: %v", err)
	}
	fmt.Printf("✓ Session status: %s\n\n", completedSession.Status)

	// Print summary
	fmt.Println("=== Integration Complete ===")
	fmt.Println("Successfully demonstrated:")
	fmt.Println("  ✓ User registration and authentication")
	fmt.Println("  ✓ Project creation and management")
	fmt.Println("  ✓ AI session creation and interaction")
	fmt.Println("  ✓ Message exchange with AI")
	fmt.Println("  ✓ Session lifecycle management")
	fmt.Println("\nAll platform services are working correctly!")

	// Optional: Save detailed output to JSON
	if len(os.Args) > 1 && os.Args[1] == "--save-json" {
		output := map[string]interface{}{
			"user":     user,
			"project":  project,
			"session":  updatedSession,
			"messages": messages.Messages,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		err := os.WriteFile("platform_integration_output.json", data, 0644)
		if err == nil {
			fmt.Println("\n✓ Saved detailed output to platform_integration_output.json")
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
