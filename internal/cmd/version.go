package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/specular/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print version information including version number, git commit,
build date, Go version, and platform.`,
	RunE: runVersion,
}

var (
	versionVerbose bool
	versionJSON    bool
)

func init() {
	versionCmd.Flags().BoolVarP(&versionVerbose, "verbose", "v", false, "show detailed version information")
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "output version information as JSON")

	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := version.GetInfo()

	// JSON output
	if versionJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Verbose output
	if versionVerbose {
		fmt.Println("\n  ╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("  ║                      [ specular ]                            ║")
		fmt.Println("  ║            AI-Native Spec and Build Assistant                ║")
		fmt.Println("  ╚══════════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println(info.String())
		return nil
	}

	// Default output (short version only)
	fmt.Printf("specular %s\n", info.Short())
	return nil
}
