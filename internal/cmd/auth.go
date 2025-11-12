package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// AuthCredentials represents stored authentication credentials
type AuthCredentials struct {
	User      string    `json:"user"`       // username or email
	Token     string    `json:"token"`      // authentication token
	ExpiresAt time.Time `json:"expires_at"` // token expiration
	Registry  string    `json:"registry"`   // registry URL
	UpdatedAt time.Time `json:"updated_at"` // last update time
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication credentials",
	Long: `Manage authentication credentials for governance, registry, and team features.

The auth command provides subcommands for logging in, logging out, checking
current authentication status, and managing tokens.

Credentials are stored securely in .specular/auth.json and include:
- User identity (username/email)
- Authentication token
- Token expiration
- Registry URL

Subcommands:
  login    Login to governance/registry
  logout   Logout and remove credentials
  whoami   Show current user identity
  token    Get or refresh authentication token

Examples:
  specular auth login --user alice@example.com
  specular auth whoami
  specular auth token --refresh
  specular auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// authLoginCmd handles user login
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to governance/registry",
	Long: `Login to governance/registry with user credentials.

This command authenticates the user and stores credentials locally
for use with governance features, policy management, and team collaboration.

The stored credentials include:
- User identity (from --user flag or prompt)
- Authentication token (from --token flag or generated)
- Registry URL (from --registry flag or default)
- Token expiration (default: 30 days)

Examples:
  specular auth login
  specular auth login --user alice@example.com
  specular auth login --user bob@example.com --registry https://registry.example.com
  specular auth login --user alice@example.com --token <token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		user, _ := cmd.Flags().GetString("user")
		token, _ := cmd.Flags().GetString("token")
		registry, _ := cmd.Flags().GetString("registry")

		// Get user identity
		if user == "" {
			user = os.Getenv("USER")
			if user == "" {
				return fmt.Errorf("--user is required (or set USER environment variable)")
			}
			fmt.Printf("Using user from environment: %s\n", user)
		}

		// Generate or use provided token
		if token == "" {
			// In a real implementation, this would authenticate with the registry
			// For now, generate a placeholder token
			token = fmt.Sprintf("token_%s_%d", user, time.Now().Unix())
			fmt.Println("Note: Using generated demo token. In production, this would authenticate with the registry.")
		}

		// Use default registry if not specified
		if registry == "" {
			registry = "https://registry.specular.dev"
		}

		// Create credentials
		creds := AuthCredentials{
			User:      user,
			Token:     token,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
			Registry:  registry,
			UpdatedAt: time.Now(),
		}

		// Save credentials
		if err := saveCredentials(creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf("✅ Logged in as: %s\n", user)
		fmt.Println()
		fmt.Printf("Registry: %s\n", registry)
		fmt.Printf("Token expires: %s\n", creds.ExpiresAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
		fmt.Println("Use 'specular auth whoami' to verify authentication.")

		return nil
	},
}

// authLogoutCmd handles user logout
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and remove credentials",
	Long: `Logout and remove stored authentication credentials.

This command removes the local authentication credentials stored in
.specular/auth.json. You will need to login again to use governance
features and team collaboration tools.

Examples:
  specular auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		authFile := filepath.Join(".specular", "auth.json")

		// Check if credentials exist
		if _, err := os.Stat(authFile); os.IsNotExist(err) {
			fmt.Println("⚠️  Not logged in.")
			return nil
		}

		// Load current credentials to show who's logging out
		creds, err := loadCredentials()
		if err == nil {
			fmt.Printf("Logging out: %s\n", creds.User)
		}

		// Remove credentials file
		if err := os.Remove(authFile); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		fmt.Println("✅ Logged out successfully.")
		fmt.Println()
		fmt.Println("Use 'specular auth login' to login again.")

		return nil
	},
}

// authWhoamiCmd shows current user identity
var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user identity",
	Long: `Show current authenticated user identity and status.

This command displays the current authentication status including:
- User identity
- Registry URL
- Token expiration
- Time since last update

Examples:
  specular auth whoami`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load credentials
		creds, err := loadCredentials()
		if err != nil {
			fmt.Println("⚠️  Not logged in.")
			fmt.Println()
			fmt.Println("Use 'specular auth login' to authenticate.")
			return nil
		}

		// Check if token is expired
		isExpired := time.Now().After(creds.ExpiresAt)

		fmt.Printf("User: %s\n", creds.User)
		fmt.Printf("Registry: %s\n", creds.Registry)
		fmt.Printf("Token expires: %s", creds.ExpiresAt.Format("2006-01-02 15:04:05"))
		if isExpired {
			fmt.Print(" (EXPIRED)")
		}
		fmt.Println()
		fmt.Printf("Last updated: %s\n", creds.UpdatedAt.Format("2006-01-02 15:04:05"))

		if isExpired {
			fmt.Println()
			fmt.Println("⚠️  Token has expired. Use 'specular auth token --refresh' to refresh.")
		}

		return nil
	},
}

// authTokenCmd manages authentication tokens
var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Get or refresh authentication token",
	Long: `Get the current authentication token or refresh it.

This command displays the current authentication token or refreshes
it if expired or near expiration.

Use --refresh to force refresh the token.
Use --show to display the full token (use with caution).

Examples:
  specular auth token                    # Show token status
  specular auth token --refresh          # Refresh the token
  specular auth token --show             # Display full token`,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		show, _ := cmd.Flags().GetBool("show")

		// Load credentials
		creds, err := loadCredentials()
		if err != nil {
			fmt.Println("⚠️  Not logged in.")
			fmt.Println()
			fmt.Println("Use 'specular auth login' to authenticate.")
			return nil
		}

		// Check if token needs refresh
		needsRefresh := time.Now().After(creds.ExpiresAt)
		nearExpiration := time.Until(creds.ExpiresAt) < 24*time.Hour

		if refresh || needsRefresh {
			// In a real implementation, this would call the registry API to refresh
			// For now, generate a new token
			fmt.Println("Refreshing token...")
			creds.Token = fmt.Sprintf("token_%s_%d", creds.User, time.Now().Unix())
			creds.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
			creds.UpdatedAt = time.Now()

			if err := saveCredentials(*creds); err != nil {
				return fmt.Errorf("failed to save refreshed token: %w", err)
			}

			fmt.Println("✅ Token refreshed successfully.")
			fmt.Println()
		}

		// Display token information
		fmt.Printf("User: %s\n", creds.User)
		fmt.Printf("Token expires: %s\n", creds.ExpiresAt.Format("2006-01-02 15:04:05"))

		if nearExpiration && !needsRefresh {
			fmt.Println()
			fmt.Println("⚠️  Token expires soon. Consider refreshing with --refresh flag.")
		}

		if show {
			fmt.Println()
			fmt.Printf("Token: %s\n", creds.Token)
			fmt.Println()
			fmt.Println("⚠️  Keep your token secure. Do not share or commit to version control.")
		} else {
			fmt.Println()
			fmt.Println("Use --show flag to display full token.")
		}

		return nil
	},
}

// loadCredentials reads credentials from .specular/auth.json
func loadCredentials() (*AuthCredentials, error) {
	authFile := filepath.Join(".specular", "auth.json")

	data, err := os.ReadFile(authFile)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var creds AuthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// saveCredentials writes credentials to .specular/auth.json
func saveCredentials(creds AuthCredentials) error {
	// Create .specular directory if it doesn't exist
	if err := os.MkdirAll(".specular", 0755); err != nil {
		return fmt.Errorf("create .specular directory: %w", err)
	}

	authFile := filepath.Join(".specular", "auth.json")

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	// Write with restricted permissions for security
	if err := os.WriteFile(authFile, data, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	return nil
}

func init() {
	// Add subcommands
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authTokenCmd)

	// Flags for login command
	authLoginCmd.Flags().String("user", "", "User identity (username or email)")
	authLoginCmd.Flags().String("token", "", "Authentication token (if already obtained)")
	authLoginCmd.Flags().String("registry", "", "Registry URL (default: https://registry.specular.dev)")

	// Flags for token command
	authTokenCmd.Flags().Bool("refresh", false, "Force refresh the token")
	authTokenCmd.Flags().Bool("show", false, "Display full token (use with caution)")

	rootCmd.AddCommand(authCmd)
}
