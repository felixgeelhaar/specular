package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/specular/internal/platform"

	"github.com/spf13/cobra"
)

var platformClient *platform.Client

// getPlatformClient returns a platform client
func getPlatformClient() *platform.Client {
	if platformClient == nil {
		apiURL := os.Getenv("SPECULAR_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:8000"
		}
		platformClient = platform.NewClient(apiURL)
	}
	return platformClient
}

// authRegisterCmd registers a new user
var authRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user account",
	Long: `Register a new user account with the Specular platform.

After registration, you will be automatically logged in.

Examples:
  specular auth register --email user@example.com --password mypass
  specular auth register --email user@example.com --password mypass --first-name John --last-name Doe`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")

		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if password == "" {
			return fmt.Errorf("--password is required")
		}
		if username == "" {
			username = email
		}
		if firstName == "" {
			firstName = "User"
		}
		if lastName == "" {
			lastName = "Account"
		}

		client := getPlatformClient()

		fmt.Printf("Registering user: %s\n", email)

		loginResp, err := client.Register(username, email, password, firstName, lastName)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		if err := savePlatformToken(loginResp.AccessToken, loginResp.RefreshToken, email); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Registration successful!")
		fmt.Printf("Email: %s\n", email)
		fmt.Println("You are now logged in.")

		return nil
	},
}

// authStatusCmd shows current auth status
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long: `Show the current authentication status and user information.

Examples:
  specular auth status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, email, err := loadPlatformToken()
		if err != nil {
			fmt.Println("Not logged in.")
			fmt.Println("Use 'specular auth login' to authenticate.")
			return nil
		}

		client := getPlatformClient()
		client.SetToken(token)

		user, err := client.GetCurrentUser()
		if err != nil {
			fmt.Println("Token may be expired or invalid.")
			fmt.Println("Use 'specular auth login' to re-authenticate.")
			return nil
		}

		fmt.Println("Logged in")
		fmt.Printf("User ID:  %s\n", user.ID)
		fmt.Printf("Email:    %s\n", email)
		fmt.Printf("Name:     %s %s\n", user.FirstName, user.LastName)
		fmt.Printf("Username: %s\n", user.Username)

		return nil
	},
}

// PlatformAuth holds platform authentication data
type PlatformAuth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
}

func savePlatformToken(accessToken, refreshToken, email string) error {
	if err := os.MkdirAll(".specular", 0755); err != nil {
		return err
	}

	auth := PlatformAuth{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Email:        email,
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(".specular", "platform_auth.json"), data, 0600)
}

func loadPlatformToken() (string, string, error) {
	data, err := os.ReadFile(filepath.Join(".specular", "platform_auth.json"))
	if err != nil {
		return "", "", err
	}

	var auth PlatformAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", "", err
	}

	return auth.AccessToken, auth.Email, nil
}

func init() {
	// Add commands to auth
	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authStatusCmd)

	// Register flags
	authRegisterCmd.Flags().String("username", "", "Username (optional)")
	authRegisterCmd.Flags().String("email", "", "Email address (required)")
	authRegisterCmd.Flags().String("password", "", "Password (required)")
	authRegisterCmd.Flags().String("first-name", "", "First name")
	authRegisterCmd.Flags().String("last-name", "", "Last name")
}
