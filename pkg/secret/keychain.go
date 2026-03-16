// Package secret provides keychain-backed secret storage for bc.
//
// Secrets are stored in the macOS Keychain using the `security` CLI tool.
// Each secret is stored as a generic password with a workspace-scoped
// service name (e.g., "bc.myproject") to isolate secrets between
// workspaces and other applications. Secret names map directly to
// environment variable names when injected into agent sessions.
package secret

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const servicePrefix = "bc"

// Store provides keychain-backed secret storage scoped to a workspace.
type Store struct {
	service string // e.g., "bc.myproject"
}

// NewStore creates a Store scoped to the given workspace name.
func NewStore(workspaceName string) *Store {
	return &Store{service: servicePrefix + "." + workspaceName}
}

// Set stores a secret in the keychain. If a secret with the same name
// already exists, it is updated.
func (s *Store) Set(ctx context.Context, name, value string) error {
	// -U updates existing or adds new
	// #nosec G204 - name is validated, value is user-provided secret
	cmd := exec.CommandContext(ctx, "security", "add-generic-password",
		"-U", "-s", s.service, "-a", name, "-w", value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set secret %q: %s", name, strings.TrimSpace(string(out)))
	}
	return nil
}

// Get retrieves a secret from the keychain.
func (s *Store) Get(ctx context.Context, name string) (string, error) {
	// #nosec G204 - name is validated by caller
	cmd := exec.CommandContext(ctx, "security", "find-generic-password",
		"-s", s.service, "-a", name, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return strings.TrimSpace(string(out)), nil
}

// Delete removes a secret from the keychain.
func (s *Store) Delete(ctx context.Context, name string) error {
	// #nosec G204 - name is validated by caller
	cmd := exec.CommandContext(ctx, "security", "delete-generic-password",
		"-s", s.service, "-a", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete secret %q: %s", name, strings.TrimSpace(string(out)))
	}
	return nil
}

// acctRegex matches "acct"<blob>="<value>" lines in security output.
var acctRegex = regexp.MustCompile(`"acct"<blob>="([^"]+)"`)

// List returns the names of all secrets stored for this workspace.
func (s *Store) List(ctx context.Context) ([]string, error) {
	// Use dump-keychain and filter for our service.
	// find-generic-password -s <svc> only returns the first match,
	// so we use dump-keychain for a complete list.
	cmd := exec.CommandContext(ctx, "security", "dump-keychain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return s.parseKeychainDump(string(out)), nil
}

// parseKeychainDump extracts account names for this store's service from
// security dump-keychain output.
func (s *Store) parseKeychainDump(output string) []string {
	var names []string

	// Split into keychain item blocks (separated by "attributes:" sections)
	blocks := strings.Split(output, "keychain: ")
	for _, block := range blocks {
		items := strings.Split(block, "attributes:")
		for _, item := range items {
			if !strings.Contains(item, fmt.Sprintf(`"svce"<blob>="%s"`, s.service)) {
				continue
			}
			matches := acctRegex.FindStringSubmatch(item)
			if len(matches) >= 2 {
				names = append(names, matches[1])
			}
		}
	}

	return names
}
