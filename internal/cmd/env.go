package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gh-curious-otter/bc/pkg/client"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

var envProvider string

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage workspace environment variables",
	Long: `Configure environment variables for agent sessions.

Environment variables are stored in settings.toml and injected into agent
sessions at startup. Use --provider to set per-provider env vars.

Priority (highest wins): agent --env file > provider env > workspace env.

Examples:
  bc env set SHARED_VAR global                           # workspace [env]
  bc env set --provider claude CLAUDE_CODE_USE_BEDROCK 1 # [providers.claude.env]
  bc env list                                            # all env vars
  bc env list --provider claude                          # claude-only env vars
  bc env get SHARED_VAR
  bc env unset SHARED_VAR
  bc env unset --provider claude CLAUDE_CODE_USE_BEDROCK`,
}

var envSetCmd = &cobra.Command{
	Use:   "set <KEY> <VALUE>",
	Short: "Set an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvSet,
}

var envGetCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvGet,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables",
	Args:  cobra.NoArgs,
	RunE:  runEnvList,
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <KEY>",
	Short: "Remove an environment variable",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvUnset,
}

func init() {
	envCmd.PersistentFlags().StringVar(&envProvider, "provider", "", "Target a specific provider (e.g., claude, gemini)")

	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envUnsetCmd)

	rootCmd.AddCommand(envCmd)
}

// getProviderEnv returns the env map for the given provider, initializing it if needed.
// Returns the ProviderConfig and an error if the provider is not found.
func getProviderEnv(cfg *workspace.Config, name string) (*workspace.ProviderConfig, error) {
	p := cfg.GetProvider(name)
	if p == nil {
		return nil, fmt.Errorf("provider %q is not configured", name)
	}
	if p.Env == nil {
		p.Env = make(map[string]string)
	}
	return p, nil
}

// envSetViaAPI sets an env var through the daemon API. Returns true if successful.
func envSetViaAPI(cmd *cobra.Command, key, value string) (bool, error) {
	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		if client.IsDaemonNotRunning(err) {
			return false, nil
		}
		return false, err
	}

	// Load current config to merge with
	raw, err := c.Settings.Get(cmd.Context())
	if err != nil {
		return false, err
	}

	var cfg workspace.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return false, fmt.Errorf("decode settings: %w", err)
	}

	if envProvider != "" {
		p, pErr := getProviderEnv(&cfg, envProvider)
		if pErr != nil {
			return false, pErr
		}
		p.Env[key] = value
		// Patch providers section
		data, mErr := json.Marshal(cfg.Providers)
		if mErr != nil {
			return false, mErr
		}
		var provPatch map[string]any
		if err := json.Unmarshal(data, &provPatch); err != nil {
			return false, err
		}
		if _, err := c.Settings.Patch(cmd.Context(), "providers", provPatch); err != nil {
			return false, err
		}
	} else {
		if cfg.Env == nil {
			cfg.Env = make(map[string]string)
		}
		cfg.Env[key] = value
		envAny := make(map[string]any, len(cfg.Env))
		for k, v := range cfg.Env {
			envAny[k] = v
		}
		if _, err := c.Settings.Patch(cmd.Context(), "env", envAny); err != nil {
			return false, err
		}
	}

	return true, nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	ok, err := envSetViaAPI(cmd, key, value)
	if err != nil {
		return err
	}
	if ok {
		if envProvider != "" {
			fmt.Printf("Set %s=%s (provider: %s)\n", key, value, envProvider)
		} else {
			fmt.Printf("Set %s=%s\n", key, value)
		}
		return nil
	}

	// Offline fallback: direct file modification
	cfg, configPath, loadErr := loadWorkspaceConfig()
	if loadErr != nil {
		return loadErr
	}

	if envProvider != "" {
		p, pErr := getProviderEnv(cfg, envProvider)
		if pErr != nil {
			return pErr
		}
		p.Env[key] = value
	} else {
		if cfg.Env == nil {
			cfg.Env = make(map[string]string)
		}
		cfg.Env[key] = value
	}

	if err := cfg.Save(configPath); err != nil {
		return err
	}

	if envProvider != "" {
		fmt.Printf("Set %s=%s (provider: %s)\n", key, value, envProvider)
	} else {
		fmt.Printf("Set %s=%s\n", key, value)
	}
	return nil
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	cfg, err := loadConfigViaAPI(cmd.Context())
	if err != nil {
		return err
	}

	if envProvider != "" {
		p := cfg.GetProvider(envProvider)
		if p == nil {
			return fmt.Errorf("provider %q is not configured", envProvider)
		}
		value, ok := p.Env[key]
		if !ok {
			return fmt.Errorf("environment variable %s is not set for provider %s", key, envProvider)
		}
		fmt.Println(value)
		return nil
	}

	value, ok := cfg.Env[key]
	if !ok {
		return fmt.Errorf("environment variable %s is not set", key)
	}

	fmt.Println(value)
	return nil
}

func runEnvList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfigViaAPI(cmd.Context())
	if err != nil {
		return err
	}

	if envProvider != "" {
		p := cfg.GetProvider(envProvider)
		if p == nil {
			return fmt.Errorf("provider %q is not configured", envProvider)
		}
		if len(p.Env) == 0 {
			fmt.Printf("No environment variables configured for provider %s\n", envProvider)
			return nil
		}
		printEnvMap(p.Env, envProvider)
		return nil
	}

	hasOutput := false

	// Workspace-level env
	if len(cfg.Env) > 0 {
		printEnvMap(cfg.Env, "")
		hasOutput = true
	}

	// Per-provider env
	for _, name := range cfg.ListProviders() {
		p := cfg.GetProvider(name)
		if p != nil && len(p.Env) > 0 {
			if hasOutput {
				fmt.Println()
			}
			printEnvMap(p.Env, name)
			hasOutput = true
		}
	}

	if !hasOutput {
		fmt.Println("No environment variables configured")
	}
	return nil
}

func printEnvMap(env map[string]string, provider string) {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if provider != "" {
		fmt.Printf("[%s]\n", provider)
	}
	for _, k := range keys {
		v := env[k]
		// Mask sensitive values (tokens, keys, secrets, passwords)
		lk := strings.ToLower(k)
		if (strings.Contains(lk, "token") || strings.Contains(lk, "key") ||
			strings.Contains(lk, "secret") || strings.Contains(lk, "password")) && len(v) > 8 {
			v = v[:4] + "****" + v[len(v)-4:]
		}
		fmt.Printf("%s=%s\n", k, v)
	}
}

func runEnvUnset(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Try API first
	c, clientErr := newDaemonClient(cmd.Context())
	if clientErr == nil {
		raw, getErr := c.Settings.Get(cmd.Context())
		if getErr != nil {
			return getErr
		}
		var cfg workspace.Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("decode settings: %w", err)
		}

		if envProvider != "" {
			p, pErr := getProviderEnv(&cfg, envProvider)
			if pErr != nil {
				return pErr
			}
			if _, ok := p.Env[key]; !ok {
				return fmt.Errorf("environment variable %s is not set for provider %s", key, envProvider)
			}
			delete(p.Env, key)
			if len(p.Env) == 0 {
				p.Env = nil
			}
			data, mErr := json.Marshal(cfg.Providers)
			if mErr != nil {
				return mErr
			}
			var provPatch map[string]any
			if err := json.Unmarshal(data, &provPatch); err != nil {
				return err
			}
			if _, err := c.Settings.Patch(cmd.Context(), "providers", provPatch); err != nil {
				return err
			}
		} else {
			if _, ok := cfg.Env[key]; !ok {
				return fmt.Errorf("environment variable %s is not set", key)
			}
			delete(cfg.Env, key)
			if len(cfg.Env) == 0 {
				cfg.Env = nil
			}
			envAny := make(map[string]any)
			if cfg.Env != nil {
				for k, v := range cfg.Env {
					envAny[k] = v
				}
			}
			if _, err := c.Settings.Patch(cmd.Context(), "env", envAny); err != nil {
				return err
			}
		}

		if envProvider != "" {
			fmt.Printf("Unset %s (provider: %s)\n", key, envProvider)
		} else {
			fmt.Printf("Unset %s\n", key)
		}
		return nil
	}

	if !client.IsDaemonNotRunning(clientErr) {
		return clientErr
	}

	// Offline fallback: direct file modification
	cfg, configPath, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	if envProvider != "" {
		p, pErr := getProviderEnv(cfg, envProvider)
		if pErr != nil {
			return pErr
		}
		if _, ok := p.Env[key]; !ok {
			return fmt.Errorf("environment variable %s is not set for provider %s", key, envProvider)
		}
		delete(p.Env, key)
		if len(p.Env) == 0 {
			p.Env = nil
		}
	} else {
		if _, ok := cfg.Env[key]; !ok {
			return fmt.Errorf("environment variable %s is not set", key)
		}
		delete(cfg.Env, key)
		if len(cfg.Env) == 0 {
			cfg.Env = nil
		}
	}

	if err := cfg.Save(configPath); err != nil {
		return err
	}

	if envProvider != "" {
		fmt.Printf("Unset %s (provider: %s)\n", key, envProvider)
	} else {
		fmt.Printf("Unset %s\n", key)
	}
	return nil
}
