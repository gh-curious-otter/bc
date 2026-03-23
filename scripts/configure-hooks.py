#!/usr/bin/env python3
"""
Configure Claude Code hooks for bc agents.

Writes .claude/settings.json with hooks that POST agent status events
to bcd via HTTP. This is the SINGLE source of truth for agent status —
no polling, no file-based IPC, no tmux scraping.

Usage:
    # Configure hooks for a tmux agent (localhost)
    python3 scripts/configure-hooks.py --workspace /path/to/workspace --runtime tmux

    # Configure hooks for a Docker agent (host.docker.internal)
    python3 scripts/configure-hooks.py --workspace /path/to/workspace --runtime docker

    # Configure with custom bcd address
    python3 scripts/configure-hooks.py --workspace /path/to/workspace --addr http://custom:9374
"""

import argparse
import json
import os
import sys

# All Claude Code hook events → agent status mapping
HOOKS = {
    # Session lifecycle
    "SessionStart": {
        "state": "idle",
        "task": "Session started",
    },
    "SessionEnd": {
        "state": "stopped",
        "task": "Session ended",
    },

    # User interaction
    "UserPromptSubmit": {
        "state": "working",
        "task": "Processing prompt...",
    },

    # Tool usage
    "PreToolUse": {
        "state": "working",
        "task_template": "Running: {tool_name}",
    },
    "PostToolUse": {
        "state": "idle",
        "task_template": "Done: {tool_name}",
    },
    "PostToolUseFailure": {
        "state": "working",
        "task_template": "Failed: {tool_name}",
    },

    # Permissions
    "PermissionRequest": {
        "state": "stuck",
        "task": "Waiting for permission",
    },

    # Response lifecycle
    "Stop": {
        "state": "idle",
        "task": "Turn complete",
    },
    "StopFailure": {
        "state": "error",
        "task": "API error",
    },

    # Notifications
    "Notification": {
        "state": "",  # no state change
        "task": "",
    },

    # Subagents
    "SubagentStart": {
        "state": "working",
        "task": "Subagent spawned",
    },
    "SubagentStop": {
        "state": "working",
        "task": "Subagent completed",
    },

    # Tasks
    "TaskCompleted": {
        "state": "done",
        "task": "Task completed",
    },

    # Teammate
    "TeammateIdle": {
        "state": "",  # no state change
        "task": "",
    },

    # Instructions
    "InstructionsLoaded": {
        "state": "",  # no state change
        "task": "",
    },

    # Config
    "ConfigChange": {
        "state": "",  # no state change
        "task": "",
    },

    # Worktree
    "WorktreeCreate": {
        "state": "starting",
        "task": "Creating worktree",
    },
    "WorktreeRemove": {
        "state": "",  # no state change
        "task": "",
    },

    # Context compaction
    "PreCompact": {
        "state": "working",
        "task": "Compacting context...",
    },
    "PostCompact": {
        "state": "working",
        "task": "Context compacted",
    },

    # MCP elicitation
    "Elicitation": {
        "state": "stuck",
        "task": "MCP input needed",
    },
    "ElicitationResult": {
        "state": "working",
        "task": "MCP input received",
    },
}


def build_hook_command(event_name: str, hook_config: dict, bcd_addr: str) -> str:
    """Build the curl command for a hook event."""
    state = hook_config.get("state", "")
    task = hook_config.get("task", "")
    task_template = hook_config.get("task_template", "")

    # Build JSON payload
    # For hooks with dynamic tool names, we use shell variable substitution
    # Claude Code exports CLAUDE_TOOL_NAME for PreToolUse/PostToolUse/PostToolUseFailure
    if task_template:
        static_task = task_template.replace("{tool_name}", "unknown")
        payload = {"event": event_name}
        if state:
            payload["state"] = state
        payload["task"] = static_task
        # Build as shell string with variable substitution
        json_body = json.dumps(payload)
        # Replace the static "unknown" with shell variable expansion
        json_body = json_body.replace("unknown", '"\'"$CLAUDE_TOOL_NAME"\'"')
    else:
        payload = {"event": event_name}
        if state:
            payload["state"] = state
        if task:
            payload["task"] = task
        json_body = json.dumps(payload)

    # Build curl command — silent, fire-and-forget, no error on failure
    cmd = (
        f"curl -sX POST {bcd_addr}/api/agents/${{BC_AGENT_ID}}/hook "
        f"-H 'Content-Type: application/json' "
        f"-d '{json_body}' "
        f"2>/dev/null || true"
    )
    return cmd


def generate_settings(bcd_addr: str) -> dict:
    """Generate the .claude/settings.json hooks section."""
    hooks = {}

    for event_name, config in HOOKS.items():
        cmd = build_hook_command(event_name, config, bcd_addr)
        hooks[event_name] = [
            {
                "hooks": [
                    {
                        "type": "command",
                        "command": cmd,
                    }
                ]
            }
        ]

    return {"hooks": hooks}


def merge_settings(existing: dict, new_hooks: dict) -> dict:
    """Merge new hooks into existing settings without clobbering user config."""
    if "hooks" not in existing:
        existing["hooks"] = {}

    for event_name, matchers in new_hooks["hooks"].items():
        if event_name not in existing["hooks"]:
            existing["hooks"][event_name] = matchers
        else:
            # Check if bc hook already exists (by checking for /api/agents in command)
            existing_cmds = []
            for matcher in existing["hooks"][event_name]:
                for hook in matcher.get("hooks", []):
                    existing_cmds.append(hook.get("command", ""))

            bc_already = any("/api/agents/" in cmd and "/hook" in cmd for cmd in existing_cmds)
            if not bc_already:
                # Append bc hook to existing matchers
                existing["hooks"][event_name].extend(matchers)

    return existing


def configure(workspace: str, bcd_addr: str) -> None:
    """Write hooks to .claude/settings.json in the workspace."""
    claude_dir = os.path.join(workspace, ".claude")
    os.makedirs(claude_dir, exist_ok=True)

    settings_path = os.path.join(claude_dir, "settings.json")
    new_hooks = generate_settings(bcd_addr)

    # Load existing settings if present
    existing = {}
    if os.path.exists(settings_path):
        try:
            with open(settings_path) as f:
                existing = json.load(f)
        except (json.JSONDecodeError, IOError):
            existing = {}

    # Merge
    merged = merge_settings(existing, new_hooks)

    # Write
    with open(settings_path, "w") as f:
        json.dump(merged, f, indent=2)
        f.write("\n")

    print(f"Configured {len(HOOKS)} hooks in {settings_path}")
    print(f"  bcd address: {bcd_addr}")
    print(f"  Events: {', '.join(HOOKS.keys())}")


def main():
    parser = argparse.ArgumentParser(
        description="Configure Claude Code hooks for bc agents"
    )
    parser.add_argument(
        "--workspace",
        default=".",
        help="Workspace root directory (default: current dir)",
    )
    parser.add_argument(
        "--runtime",
        choices=["tmux", "docker"],
        default="tmux",
        help="Agent runtime — determines bcd address (default: tmux)",
    )
    parser.add_argument(
        "--addr",
        default="",
        help="Custom bcd address (overrides --runtime)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print settings JSON without writing",
    )

    args = parser.parse_args()

    # Determine bcd address
    if args.addr:
        bcd_addr = args.addr
    elif args.runtime == "docker":
        bcd_addr = "http://host.docker.internal:9374"
    else:
        bcd_addr = "http://127.0.0.1:9374"

    if args.dry_run:
        settings = generate_settings(bcd_addr)
        print(json.dumps(settings, indent=2))
        return

    workspace = os.path.abspath(args.workspace)
    if not os.path.isdir(workspace):
        print(f"Error: workspace {workspace} does not exist", file=sys.stderr)
        sys.exit(1)

    configure(workspace, bcd_addr)


if __name__ == "__main__":
    main()
