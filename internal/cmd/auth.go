package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Note: The logout command uses loadPlatformToken from auth_platform.go

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication credentials",
	Long: `Manage authentication credentials for the Specular platform.

The auth command provides subcommands for registering, logging in, logging out,
and checking current authentication status.

Credentials are stored securely in .specular/platform_auth.json.

Subcommands:
  register  Register a new user account
  login     Login with email and password
  logout    Logout and remove credentials
  status    Show current authentication status

Examples:
  specular auth register --email user@example.com --password mypass
  specular auth login --email user@example.com --password mypass
  specular auth status
  specular auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// authLoginCmd handles user login
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the platform",
	Long: `Login to the Specular platform with your email and password.

After logging in, your access token will be saved locally.

Examples:
  specular auth login --email user@example.com --password mypass`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if password == "" {
			return fmt.Errorf("--password is required")
		}

		client := getPlatformClient()

		fmt.Printf("Logging in as: %s\n", email)

		loginResp, err := client.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := savePlatformToken(loginResp.AccessToken, loginResp.RefreshToken, email); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Login successful!")

		return nil
	},
}

// authLogoutCmd handles user logout
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and remove credentials",
	Long: `Logout and remove stored authentication credentials.

This command removes the local authentication credentials stored in
.specular/platform_auth.json. You will need to login again to use
platform features.

Examples:
  specular auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		authFile := filepath.Join(".specular", "platform_auth.json")

		// Check if credentials exist
		if _, err := os.Stat(authFile); os.IsNotExist(err) {
			fmt.Println("Not logged in.")
			return nil
		}

		// Load current credentials to show who's logging out
		_, email, err := loadPlatformToken()
		if err == nil {
			fmt.Printf("Logging out: %s\n", email)
		}

		// Remove credentials file
		if err := os.Remove(authFile); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		fmt.Println("Logged out successfully.")
		fmt.Println()
		fmt.Println("Use 'specular auth login' to login again.")

		return nil
	},
}

func init() {
	// Add subcommands
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)

	// Flags for login command
	authLoginCmd.Flags().String("email", "", "Email address (required)")
	authLoginCmd.Flags().String("password", "", "Password (required)")

	rootCmd.AddCommand(authCmd)
}
