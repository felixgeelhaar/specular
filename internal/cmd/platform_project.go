package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var platformProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long: `Manage projects on the Specular platform.

Projects organize AI sessions and provide a workspace for collaboration.

Subcommands:
  list    List all projects
  create  Create a new project
  show    Show project details
  delete  Delete a project

Examples:
  specular platform project list
  specular platform project create --name "My Project" --description "A cool project"
  specular platform project show <project-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var platformProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		projects, err := client.ListProjects(page, pageSize)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects.Projects) == 0 {
			fmt.Println("No projects found.")
			fmt.Println("\nCreate one with: specular platform project create --name \"My Project\"")
			return nil
		}

		fmt.Printf("Projects (page %d):\n\n", projects.Page)
		for _, p := range projects.Projects {
			fmt.Printf("ID:          %s\n", p.ID)
			fmt.Printf("Name:        %s\n", p.Name)
			if p.Description != "" {
				fmt.Printf("Description: %s\n", p.Description)
			}
			fmt.Printf("Status:      %s\n", p.Status)
			fmt.Printf("Created:     %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Println("---")
		}

		return nil
	},
}

var platformProjectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		visibility, _ := cmd.Flags().GetString("visibility")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		project, err := client.CreateProject(name, description, visibility, nil)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		fmt.Println("Project created!")
		fmt.Printf("ID:          %s\n", project.ID)
		fmt.Printf("Name:        %s\n", project.Name)
		if project.Description != "" {
			fmt.Printf("Description: %s\n", project.Description)
		}
		fmt.Printf("Status:      %s\n", project.Status)

		return nil
	},
}

var platformProjectShowCmd = &cobra.Command{
	Use:   "show <project-id>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		project, err := client.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		fmt.Printf("ID:          %s\n", project.ID)
		fmt.Printf("Name:        %s\n", project.Name)
		if project.Description != "" {
			fmt.Printf("Description: %s\n", project.Description)
		}
		fmt.Printf("Status:      %s\n", project.Status)
		fmt.Printf("Owner:       %s\n", project.OwnerID)
		fmt.Printf("Created:     %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var platformProjectDeleteCmd = &cobra.Command{
	Use:   "delete <project-id>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete project %s? (use --force to confirm)\n", projectID)
			return nil
		}

		token, _, err := loadPlatformToken()
		if err != nil {
			return fmt.Errorf("not logged in - run 'specular auth login' first")
		}

		client := getPlatformClient()
		client.SetToken(token)

		if err := client.DeleteProject(projectID); err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		fmt.Printf("Project deleted: %s\n", projectID)

		return nil
	},
}

func init() {
	// Project subcommands
	platformProjectCmd.AddCommand(platformProjectListCmd)
	platformProjectCmd.AddCommand(platformProjectCreateCmd)
	platformProjectCmd.AddCommand(platformProjectShowCmd)
	platformProjectCmd.AddCommand(platformProjectDeleteCmd)

	// Flags for list command
	platformProjectListCmd.Flags().Int("page", 1, "Page number")
	platformProjectListCmd.Flags().Int("page-size", 10, "Items per page")

	// Flags for create command
	platformProjectCreateCmd.Flags().String("name", "", "Project name (required)")
	platformProjectCreateCmd.Flags().String("description", "", "Project description")
	platformProjectCreateCmd.Flags().String("visibility", "private", "Visibility (private/public)")

	// Flags for delete command
	platformProjectDeleteCmd.Flags().Bool("force", false, "Force deletion without confirmation")

	// Add project to platform
	platformCmd.AddCommand(platformProjectCmd)
}
