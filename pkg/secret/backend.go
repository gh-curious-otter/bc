package secret

// Backend is the storage interface for encrypted secrets persistence.
// Store is the default SQLite implementation.
type Backend interface {
	// Set creates or updates a secret with an encrypted value.
	Set(name, value, description string) error
	// GetValue retrieves and decrypts a secret value.
	GetValue(name string) (string, error)
	// GetMeta returns metadata for a secret (no value).
	GetMeta(name string) (*SecretMeta, error)
	// List returns metadata for all secrets (no values).
	List() ([]*SecretMeta, error)
	// Delete removes a secret.
	Delete(name string) error
	// ResolveEnv resolves ${secret:NAME} references in an env map.
	ResolveEnv(env map[string]string) map[string]string
	// Close releases database resources.
	Close() error
}
