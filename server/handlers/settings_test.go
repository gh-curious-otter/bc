package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// newTestWorkspace creates a minimal workspace in a temp directory with a valid config.
func newTestWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := &workspace.Config{
		Workspace: workspace.WorkspaceConfig{
			Name:    "test-ws",
			Version: workspace.ConfigVersion,
		},
		Providers: workspace.ProvidersConfig{
			Default: "claude",
			Claude:  &workspace.ProviderConfig{Enabled: true},
		},
		Runtime: workspace.RuntimeConfig{
			Backend: "tmux",
		},
	}
	return &workspace.Workspace{
		Config:  cfg,
		RootDir: dir,
	}
}

func TestSettingsPatchSection(t *testing.T) {
	ws := newTestWorkspace(t)
	h := NewSettingsHandler(ws)

	mux := http.NewServeMux()
	h.Register(mux)

	tests := []struct {
		body       string
		wantErr    string
		name       string
		section    string
		wantStatus int
	}{
		{
			name:       "patch user section",
			section:    "user",
			body:       `{"nickname":"@alice"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "patch runtime section",
			section:    "runtime",
			body:       `{"backend":"docker"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "patch env section",
			section:    "env",
			body:       `{"FOO":"bar"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "patch logs section",
			section:    "logs",
			body:       `{"path":"custom/logs"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown section returns 400",
			section:    "bogus",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "unknown section: bogus",
		},
		{
			name:       "invalid JSON returns 400",
			section:    "user",
			body:       `{not json}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid user config:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/settings/"+tt.section, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantErr != "" {
				if !strings.Contains(rec.Body.String(), tt.wantErr) {
					t.Errorf("body = %q, want to contain %q", rec.Body.String(), tt.wantErr)
				}
			}
		})
	}
}

func TestSettingsPatchUpdatesConfig(t *testing.T) {
	ws := newTestWorkspace(t)
	h := NewSettingsHandler(ws)

	mux := http.NewServeMux()
	h.Register(mux)

	// PATCH user section
	body := `{"nickname":"@bob"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/settings/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify the response contains the full config with updated user.
	var result map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify the in-memory config was updated.
	if ws.Config.User.Nickname != "@bob" {
		t.Errorf("config.User.Nickname = %q, want %q", ws.Config.User.Nickname, "@bob")
	}
}

func TestSettingsPatchMethodNotAllowed(t *testing.T) {
	ws := newTestWorkspace(t)
	h := NewSettingsHandler(ws)

	mux := http.NewServeMux()
	h.Register(mux)

	// GET on /api/settings/user should be 405
	req := httptest.NewRequest(http.MethodGet, "/api/settings/user", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSettingsPatchAllSections(t *testing.T) {
	ws := newTestWorkspace(t)
	h := NewSettingsHandler(ws)

	mux := http.NewServeMux()
	h.Register(mux)

	sections := []struct {
		name string
		body string
	}{
		{"user", `{"nickname":"@test"}`},
		{"tui", `{}`},
		{"runtime", `{"backend":"tmux"}`},
		{"providers", `{"default":"claude"}`},
		{"services", `{}`},
		{"logs", `{}`},
		{"performance", `{}`},
		{"env", `{}`},
		{"roster", `{}`},
	}

	for _, s := range sections {
		t.Run(s.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/settings/"+s.name, strings.NewReader(s.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("section %q: status = %d, want %d; body = %s", s.name, rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

func TestSettingsPatchNilConfig(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Config = nil
	h := NewSettingsHandler(ws)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/settings/user", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
