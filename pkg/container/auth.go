package container

import (
	"path/filepath"
)

// AgentVolumeDir returns the persistent volume directory for an agent's Claude state.
// Mounted as /home/agent/.claude inside the container. Starts empty on first
// create — Claude populates it with auth, plugins, settings, sessions.
//
// Layout: <workspaceDir>/.bc/volumes/<agentName>/.claude
func AgentVolumeDir(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", "volumes", agentName, ".claude")
}
