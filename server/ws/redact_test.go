package ws

import (
	"strings"
	"testing"
)

func TestRedactSecrets_GitHubPAT(t *testing.T) {
	input := `GH_TOKEN="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" gh pr create`
	result := RedactSecrets(input)
	if strings.Contains(result, "github_pat_") {
		t.Errorf("GitHub PAT not redacted: %s", result)
	}
	if !strings.Contains(result, "***") {
		t.Error("expected *** in redacted output")
	}
}

func TestRedactSecrets_GHP(t *testing.T) {
	input := "Authorization: token ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"
	result := RedactSecrets(input)
	if strings.Contains(result, "ghp_") {
		t.Errorf("ghp token not redacted: %s", result)
	}
}

func TestRedactSecrets_AWSKey(t *testing.T) {
	input := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	result := RedactSecrets(input)
	if strings.Contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key not redacted: %s", result)
	}
}

func TestRedactSecrets_NoSecrets(t *testing.T) {
	input := "go build ./..."
	result := RedactSecrets(input)
	if result != input {
		t.Errorf("clean string was modified: %q -> %q", input, result)
	}
}

func TestRedactMap_NestedSecrets(t *testing.T) {
	data := map[string]any{
		"command": `GH_TOKEN="github_pat_11CAICWNY09vzYZ4KAovm5_test" gh pr list`,
		"agent":   "swift-hawk",
		"nested": map[string]any{
			"token": "should-be-redacted-by-key",
		},
	}
	result := RedactMap(data)

	cmd, ok := result["command"].(string)
	if !ok {
		t.Fatal("command not a string")
	}
	if strings.Contains(cmd, "github_pat_") {
		t.Errorf("command not redacted: %s", cmd)
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested not a map")
	}
	if nested["token"] != "***" {
		t.Errorf("nested token not redacted: %v", nested["token"])
	}

	// Non-secret field should be preserved
	if result["agent"] != "swift-hawk" {
		t.Errorf("agent field modified: %v", result["agent"])
	}
}

func TestRedactMap_Nil(t *testing.T) {
	result := RedactMap(nil)
	if result != nil {
		t.Error("nil map should return nil")
	}
}
