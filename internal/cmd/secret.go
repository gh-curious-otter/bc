package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/secret"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage workspace secrets",
	Long: `Store and manage secrets in the macOS Keychain.

Secrets are scoped to the current workspace and injected as environment
variables when agents start.

Examples:
  bc secret create AWS_BEARER_TOKEN_BEDROCK    # Prompt for value
  bc secret create API_KEY --from-env          # Import from env var
  bc secret list                               # List secret names
  bc secret delete AWS_BEARER_TOKEN_BEDROCK    # Remove a secret`,
}

var secretCreateCmd = &cobra.Command{
	Use:   "create <NAME>",
	Short: "Create or update a secret",
	Long: `Store a secret in the macOS Keychain for the current workspace.

The secret name should match the environment variable name you want
injected into agent sessions (e.g., AWS_BEARER_TOKEN_BEDROCK).

By default, prompts for the value with hidden input. Use --from-env
to import the value from an existing environment variable of the same name.

Examples:
  bc secret create AWS_BEARER_TOKEN_BEDROCK
  bc secret create API_KEY --from-env`,
	Args: cobra.ExactArgs(1),
	RunE: runSecretCreate,
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secret names",
	Args:  cobra.NoArgs,
	RunE:  runSecretList,
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete <NAME>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretDelete,
}

var secretFromEnv bool

func init() {
	secretCreateCmd.Flags().BoolVar(&secretFromEnv, "from-env", false, "Import value from environment variable of the same name")

	secretCmd.AddCommand(secretCreateCmd)
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretDeleteCmd)

	rootCmd.AddCommand(secretCmd)
}

func getSecretStore() (*secret.Store, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}
	return secret.NewStore(ws.Name()), nil
}

func runSecretCreate(_ *cobra.Command, args []string) error {
	store, err := getSecretStore()
	if err != nil {
		return err
	}

	ctx := context.Background()
	name := args[0]
	var value string

	if secretFromEnv {
		value = os.Getenv(name)
		if value == "" {
			return fmt.Errorf("environment variable %s is not set or empty", name)
		}
	} else {
		fmt.Printf("Enter value for %s: ", name)
		raw, readErr := term.ReadPassword(os.Stdin.Fd())
		fmt.Println() // newline after hidden input
		if readErr != nil {
			return fmt.Errorf("failed to read secret: %w", readErr)
		}
		value = string(raw)
		if value == "" {
			return fmt.Errorf("secret value cannot be empty")
		}
	}

	if err := store.Set(ctx, name, value); err != nil {
		return err
	}

	fmt.Printf("Secret %s stored\n", name)
	return nil
}

func runSecretList(_ *cobra.Command, _ []string) error {
	store, err := getSecretStore()
	if err != nil {
		return err
	}

	names, err := store.List(context.Background())
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No secrets stored for this workspace")
		return nil
	}

	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

func runSecretDelete(_ *cobra.Command, args []string) error {
	store, err := getSecretStore()
	if err != nil {
		return err
	}

	if err := store.Delete(context.Background(), args[0]); err != nil {
		return err
	}

	fmt.Printf("Secret %s deleted\n", args[0])
	return nil
}
