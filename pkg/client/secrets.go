package client

import (
	"context"
	"time"
)

// SecretsClient provides secret operations via the daemon.
type SecretsClient struct {
	client *Client
}

// SecretInfo represents secret metadata returned by the daemon.
// Values are never exposed — only metadata.
type SecretInfo struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
}

// List returns all secret metadata from the daemon.
func (s *SecretsClient) List(ctx context.Context) ([]SecretInfo, error) {
	var secrets []SecretInfo
	if err := s.client.get(ctx, "/api/secrets", &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

// Create creates a new secret.
func (s *SecretsClient) Create(ctx context.Context, name, value string) (*SecretInfo, error) {
	body := map[string]string{"name": name, "value": value}
	var info SecretInfo
	if err := s.client.post(ctx, "/api/secrets", body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Get returns metadata for a single secret by name.
func (s *SecretsClient) Get(ctx context.Context, name string) (*SecretInfo, error) {
	var info SecretInfo
	if err := s.client.get(ctx, "/api/secrets/"+name, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Update updates an existing secret's value.
func (s *SecretsClient) Update(ctx context.Context, name, value string) (*SecretInfo, error) {
	body := map[string]string{"value": value}
	var info SecretInfo
	if err := s.client.put(ctx, "/api/secrets/"+name, body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Delete removes a secret by name.
func (s *SecretsClient) Delete(ctx context.Context, name string) error {
	return s.client.delete(ctx, "/api/secrets/"+name)
}
