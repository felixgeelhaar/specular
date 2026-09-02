package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/worktree"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage Git worktrees for parallel agent sessions",
	Long: `Create and manage Git worktrees so multiple Specular (or external coding-agent)
sessions can work on the same repository without colliding.

Each managed worktree lives under .specular/worktrees/<name> on a branch
named specular/<name>. Worktree path and branch are recorded in attestation
provenance when used with 'specular auto --worktree'.

This is the Specular-side answer to the parallel-session isolation pattern
used by agentic development environments — with the governance difference
that the worktree identity flows into the audit chain.

Examples:
  specular worktree create fix-auth
  specular worktree list
  specular worktree remove fix-auth --delete-branch
  specular auto --worktree parallel-1 "Add health check endpoint"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var worktreeCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create an isolated worktree and branch",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := worktree.NewManager(cwd)
		if err != nil {
			return err
		}

		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		branch, _ := cmd.Flags().GetString("branch")
		base, _ := cmd.Flags().GetString("base")
		force, _ := cmd.Flags().GetBool("force")
		jsonOut, _ := cmd.Flags().GetBool("json")

		info, err := mgr.Create(cmd.Context(), worktree.Options{
			Name:   name,
			Branch: branch,
			Base:   base,
			Force:  force,
		})
		if err != nil {
			return err
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		fmt.Printf("Created worktree %s\n", info.Name)
		fmt.Printf("  Path:   %s\n", info.Path)
		fmt.Printf("  Branch: %s\n", info.Branch)
		if info.Head != "" {
			fmt.Printf("  HEAD:   %s\n", shortSHA(info.Head))
		}
		fmt.Printf("\nNext: cd %s && specular auto \"your goal\"\n", info.Path)
		fmt.Printf("  or: specular auto --worktree %s \"your goal\"\n", info.Name)
		return nil
	},
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Git worktrees for this repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := worktree.NewManager(cwd)
		if err != nil {
			return err
		}

		managedOnly, _ := cmd.Flags().GetBool("managed")
		jsonOut, _ := cmd.Flags().GetBool("json")

		list, err := mgr.List(cmd.Context())
		if err != nil {
			return err
		}

		if managedOnly {
			filtered := list[:0]
			for _, w := range list {
				if w.Managed {
					filtered = append(filtered, w)
				}
			}
			list = filtered
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(list)
		}

		if len(list) == 0 {
			fmt.Println("No worktrees found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tBRANCH\tMANAGED\tPATH")
		for _, info := range list {
			managed := "no"
			if info.Managed {
				managed = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, info.Branch, managed, info.Path)
		}
		return w.Flush()
	},
}

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <name-or-path>",
	Short: "Remove a worktree (optionally delete its branch)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		mgr, err := worktree.NewManager(cwd)
		if err != nil {
			return err
		}

		target := args[0]
		deleteBranch, _ := cmd.Flags().GetBool("delete-branch")

		// Resolve name to path if needed.
		path := target
		if !isAbsOrDotPath(target) {
			candidate := ""
			list, listErr := mgr.List(cmd.Context())
			if listErr == nil {
				for _, w := range list {
					if w.Name == target || w.Branch == worktree.BranchPrefix+target {
						candidate = w.Path
						break
					}
				}
			}
			if candidate == "" {
				candidate = filepath.Join(mgr.RepoRoot(), worktree.DefaultRelativeDir, target)
			}
			path = candidate
		}

		if err := mgr.Remove(cmd.Context(), path, deleteBranch); err != nil {
			return err
		}
		fmt.Printf("Removed worktree %s\n", path)
		return nil
	},
}

func isAbsOrDotPath(s string) bool {
	return len(s) > 0 && (s[0] == '/' || s[0] == '.' || (len(s) > 1 && s[1] == ':'))
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func init() {
	worktreeCreateCmd.Flags().String("branch", "", "Branch name (default: specular/<name>)")
	worktreeCreateCmd.Flags().String("base", "HEAD", "Starting ref for the new branch")
	worktreeCreateCmd.Flags().Bool("force", false, "Replace an existing worktree directory")
	worktreeCreateCmd.Flags().Bool("json", false, "Emit JSON")

	worktreeListCmd.Flags().Bool("managed", false, "Show only Specular-managed worktrees")
	worktreeListCmd.Flags().Bool("json", false, "Emit JSON")

	worktreeRemoveCmd.Flags().Bool("delete-branch", false, "Also delete the specular/* branch")

	worktreeCmd.AddCommand(worktreeCreateCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
	rootCmd.AddCommand(worktreeCmd)
}
