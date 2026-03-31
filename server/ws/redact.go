package ws

import (
	"regexp"
	"strings"
)

// secretPatterns matches known secret/token formats that should be redacted.
var secretPatterns = []*regexp.Regexp{
	// GitHub PATs (classic and fine-grained)
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`gho_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`ghs_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`ghr_[A-Za-z0-9]{36,}`),

	// AWS
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key["'=:\s]+[A-Za-z0-9/+=]{20,}`),

	// Generic secret patterns in env var assignments
	regexp.MustCompile(`(?i)(GH_TOKEN|GITHUB_TOKEN|GITHUB_PERSONAL_ACCESS_TOKEN|AWS_SECRET_ACCESS_KEY|AWS_ACCESS_KEY_ID|CLOUDFLARE_API_TOKEN|TELEGRAM_BOT_TOKEN|SLACK_BOT_TOKEN|SLACK_APP_TOKEN|DISCORD_BOT_TOKEN|DATABASE_URL|STATS_DATABASE_URL)["'=:\s]+[^\s"']{8,}`),

	// Bearer tokens
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{20,}`),

	// Generic long hex/base64 tokens (likely secrets)
	regexp.MustCompile(`(?i)(token|secret|password|api[_-]?key)["'=:\s]+[A-Za-z0-9._\-/+=]{20,}`),
}

const redacted = "***"

// RedactSecrets replaces known secret patterns in a string with "***".
func RedactSecrets(s string) string {
	for _, pat := range secretPatterns {
		s = pat.ReplaceAllString(s, redacted)
	}
	return s
}

// RedactMap recursively redacts secrets from string values in a map.
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = redactValue(k, v)
	}
	return result
}

func redactValue(key string, v any) any {
	// Redact entire value if key looks like a secret
	lk := strings.ToLower(key)
	if strings.Contains(lk, "token") || strings.Contains(lk, "secret") ||
		strings.Contains(lk, "password") || strings.Contains(lk, "api_key") ||
		strings.Contains(lk, "apikey") {
		if s, ok := v.(string); ok && len(s) > 0 {
			return redacted
		}
	}

	switch val := v.(type) {
	case string:
		return RedactSecrets(val)
	case map[string]any:
		return RedactMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = redactValue("", item)
		}
		return out
	default:
		return v
	}
}
