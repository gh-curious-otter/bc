// Agent comment formatting for consistent **[agent]** prefix (#292).
package channel

// FormatAgentComment returns a comment body with the standard **[agent]** prefix
// so TUI/UI can style or fold agent-generated comments. Use when bc or agents
// post comments (channels, GitHub, etc.).
func FormatAgentComment(agentID, body string) string {
	if agentID == "" {
		return body
	}
	prefix := "**[" + agentID + "]**"
	if body == "" {
		return prefix
	}
	return prefix + " " + body
}
