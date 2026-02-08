package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/github"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration",
	Long:  `Commands for GitHub authentication and repository operations.`,
}

var githubAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "GitHub authentication",
	Long:  `Login, status, and token storage info for GitHub (via gh CLI).`,
}

var githubAuthLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to GitHub",
	Long: `Run 'gh auth login' to authenticate with GitHub interactively.

For scripted token use: gh auth login --with-token < token.txt`,
	Args: cobra.NoArgs,
	RunE: runGitHubAuthLogin,
}

var githubAuthStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GitHub auth status",
	Long:  `Show whether you are logged in to GitHub and which account is active.`,
	Args:  cobra.NoArgs,
	RunE:  runGitHubAuthStatus,
}

func init() {
	githubAuthCmd.AddCommand(githubAuthLoginCmd)
	githubAuthCmd.AddCommand(githubAuthStatusCmd)
	githubCmd.AddCommand(githubAuthCmd)
	rootCmd.AddCommand(githubCmd)
}

func runGitHubAuthLogin(cmd *cobra.Command, args []string) error {
	if err := github.AuthLogin(); err != nil {
		return fmt.Errorf("gh auth login: %w", err)
	}
	fmt.Println("Login completed. Run 'bc github auth status' to verify.")
	return nil
}

func runGitHubAuthStatus(cmd *cobra.Command, args []string) error {
	result, err := github.AuthStatus()
	if err != nil {
		return fmt.Errorf("github auth status: %w", err)
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if !result.LoggedIn {
		fmt.Println("Not logged in to GitHub.")
		fmt.Println()
		fmt.Println(github.TokenStorageInfo())
		fmt.Println()
		fmt.Println("Run 'bc github auth login' to log in.")
		return nil
	}

	fmt.Println("Logged in to GitHub")
	for _, a := range result.Accounts {
		active := ""
		if a.Active {
			active = " (active)"
		}
		fmt.Printf("  %s: %s%s\n", a.Host, a.Login, active)
		fmt.Printf("    Token: %s\n", a.TokenSource)
		fmt.Printf("    Scopes: %s\n", a.Scopes)
	}
	fmt.Println()
	fmt.Println(github.TokenStorageInfo())
	return nil
}
