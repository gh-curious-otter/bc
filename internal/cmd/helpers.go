package cmd

import "regexp"

// identifierPattern matches valid identifiers: starts with a letter or underscore,
// followed by letters, numbers, dashes, or underscores.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// validIdentifier checks if a string is a valid identifier (alphanumeric, dash, underscore).
// Must start with letter or underscore, then letters, numbers, dashes, underscores.
func validIdentifier(s string) bool {
	if s == "" {
		return false
	}
	return identifierPattern.MatchString(s)
}
