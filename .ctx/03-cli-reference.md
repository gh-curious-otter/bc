# Gas Town CLI Reference

The `gt` command-line interface provides comprehensive control over Gas Town workspaces, agents, and workflows.

## Command Categories

| Category | Commands | Purpose |
|----------|----------|---------|
| **Work Management** | sling, convoy, done, hook, mq | Assign, track, and complete work |
| **Agent Operations** | agents, polecat, witness, refinery, deacon, mayor, prime, attach | Manage agent lifecycle |
| **Communication** | mail, nudge, broadcast, handoff | Inter-agent messaging |
| **Services** | up, down, daemon, boot, feed | Control infrastructure |
| **Workspace** | rig, crew, init, install | Manage workspaces and workers |
| **Configuration** | config, role, plugin, theme | System settings |
| **Diagnostics** | doctor, status, log, costs, activity | Health checks and monitoring |

---

## Work Management

### gt sling

Assign work to an agent by attaching it to their hook.

```bash
gt sling <bead-id> <target>
gt sling gt-abc123 gastown/toast       # Assign to polecat
gt sling gt-abc123 gastown/crew/max    # Assign to crew member
gt sling gt-abc123 gastown/witness     # Assign to witness
```

**Flags:**
- `--nudge`, `-n` - Send notification after slinging
- `--message`, `-m <text>` - Include message with the assignment

**Behavior:**
1. Validates the bead exists
2. Pins the bead to the target agent's hook
3. Optionally nudges the agent to start work

### gt convoy

Orchestrate multi-step workflows with dependencies.

```bash
gt convoy create <name> --steps "step1,step2,step3"
gt convoy status <convoy-id>
gt convoy cancel <convoy-id>
```

**Use Cases:**
- Sequential task execution
- Parallel task fan-out
- Dependency-based workflows

### gt done

Mark work as complete and release the hook.

```bash
gt done                          # Complete current work
gt done --exit COMPLETE          # Exit with completion status
gt done --exit DEFERRED          # Defer work for later
gt done --status blocked         # Mark as blocked
gt done --message "Finished X"   # Add completion message
```

**Exit Types:**
- `COMPLETE` - Work finished successfully
- `DEFERRED` - Work paused, will resume later
- `BLOCKED` - Waiting on external dependency
- `FAILED` - Work could not be completed

**Polecat Behavior:**
- Polecats use `gt done` to signal completion
- Witness handles cleanup and potential respawn

### gt hook

View and manage the current agent's work hook.

```bash
gt hook                    # Show what's on the hook
gt hook show               # Detailed view of hooked work
gt hook clear              # Clear the hook (unsling)
gt hook --json             # JSON output
```

**The Hook Concept:**
- Each agent has exactly one hook
- Work is "pinned" to the hook when assigned
- Hook represents the agent's current focus
- Prevents agents from juggling multiple tasks

### gt mq

Manage the merge queue (Refinery work queue).

```bash
gt mq list                 # List queued merge requests
gt mq status               # Show queue health
gt mq add <branch>         # Add branch to merge queue
gt mq prioritize <id>      # Move item to front
gt mq remove <id>          # Remove from queue
```

**Queue States:**
- `idle` - No work being processed
- `processing` - Refinery actively merging
- `blocked` - Items waiting on dependencies

---

## Agent Operations

### gt agents

List all running agent sessions.

```bash
gt agents                  # List all agents
gt agents --rig gastown    # Filter by rig
gt agents --type polecat   # Filter by type
gt agents --json           # JSON output
gt agents --running        # Only running agents
```

**Agent Types:**
- `mayor` - Global work coordinator
- `deacon` - Health orchestrator
- `witness` - Per-rig polecat manager
- `refinery` - Per-rig merge processor
- `polecat` - Ephemeral worker
- `crew` - Persistent human workspace

### gt polecat

Manage ephemeral polecat workers.

```bash
gt polecat list                           # List polecats
gt polecat spawn <name> --rig gastown     # Create polecat
gt polecat stop <name>                    # Stop polecat
gt polecat stopall                        # Stop all polecats
gt polecat logs <name>                    # View logs
```

**Polecat Lifecycle:**
1. Spawned by Witness when work is available
2. Receives work assignment via sling
3. Executes work autonomously
4. Calls `gt done` when complete
5. Witness handles cleanup

### gt witness

Control the Witness agent for a rig.

```bash
gt witness start --rig gastown    # Start witness
gt witness stop --rig gastown     # Stop witness
gt witness status --rig gastown   # Check status
gt witness restart --rig gastown  # Restart
```

**Witness Responsibilities:**
- Monitor work queue for the rig
- Spawn polecats when work is ready
- Track polecat health and progress
- Handle polecat completion/failure

### gt refinery

Control the Refinery agent for a rig.

```bash
gt refinery start --rig gastown   # Start refinery
gt refinery stop --rig gastown    # Stop refinery
gt refinery status --rig gastown  # Check status
gt refinery queue --rig gastown   # View merge queue
```

**Refinery Responsibilities:**
- Process merge requests
- Validate changes
- Execute merge operations
- Handle merge conflicts

### gt deacon

Manage the Deacon health orchestrator.

```bash
gt deacon start              # Start deacon
gt deacon stop               # Stop deacon
gt deacon status             # Check status
gt deacon restart            # Restart
```

**Deacon Responsibilities:**
- Monitor Mayor and Witness health
- Handle agent failures
- Orchestrate restarts
- Manage Boot watchdog

### gt mayor

Control the Mayor global coordinator.

```bash
gt mayor start               # Start mayor
gt mayor stop                # Stop mayor
gt mayor status              # Check status
gt mayor restart             # Restart
```

**Mayor Responsibilities:**
- Global work coordination
- Cross-rig orchestration
- Priority management
- Escalation handling

### gt prime

Output role context for the current directory.

```bash
gt prime                     # Detect role and output context
gt prime --hook              # Hook mode for session startup
gt prime --state             # Show session state only
gt prime --dry-run           # Preview without side effects
gt prime --explain           # Show why sections were included
```

**Role Detection:**
- Town root / mayor/ -> Mayor context
- `<rig>/witness/` -> Witness context
- `<rig>/refinery/rig/` -> Refinery context
- `<rig>/polecats/<name>/` -> Polecat context
- `<rig>/crew/<name>/` -> Crew context

**Hook Mode:**
Used by Claude Code SessionStart hook:
```json
{"hooks": [{"type": "command", "command": "gt prime --hook"}]}
```

### gt attach

Attach to an agent's tmux session.

```bash
gt attach <target>                    # Attach to agent
gt attach gastown/toast               # Attach to polecat
gt attach gastown/witness             # Attach to witness
gt attach mayor                       # Attach to mayor
```

---

## Communication

### gt mail

Agent messaging system.

#### Send Messages

```bash
gt mail send <address> -s "Subject" -m "Body"
gt mail send gastown/toast -s "Status" -m "How's the bug fix?"
gt mail send mayor/ -s "Complete" -m "Finished gt-abc"
gt mail send --self -s "Handoff" -m "Context for next session"
```

**Flags:**
- `-s, --subject <text>` - Message subject (required)
- `-m, --message <text>` - Message body
- `--type <type>` - Message type (task, scavenge, notification, reply)
- `--priority <0-4>` - Priority (0=urgent, 2=normal, 4=backlog)
- `--urgent` - Set priority=0
- `--notify` - Send tmux notification
- `--reply-to <id>` - Reply to message
- `--cc <address>` - CC recipients
- `--self` - Send to self

**Address Formats:**
- `mayor/` - Mayor inbox
- `<rig>/witness` - Rig's Witness
- `<rig>/refinery` - Rig's Refinery
- `<rig>/<polecat>` - Polecat
- `<rig>/crew/<name>` - Crew worker

#### Check Inbox

```bash
gt mail inbox                  # Current context inbox
gt mail inbox --unread         # Unread only
gt mail inbox mayor/           # Mayor's inbox
gt mail inbox --json           # JSON output
```

#### Read Messages

```bash
gt mail read <message-id>      # Read specific message
gt mail thread <thread-id>     # View conversation thread
gt mail peek                   # Preview first unread
```

#### Manage Messages

```bash
gt mail mark-read <id>         # Mark as read
gt mail mark-unread <id>       # Mark as unread
gt mail archive <id>           # Archive message
gt mail delete <id>            # Delete message
gt mail clear                  # Clear all messages
```

#### Reply

```bash
gt mail reply <message-id> -m "Response text"
gt mail reply msg-abc123 -s "Custom subject" -m "Reply body"
```

#### Search

```bash
gt mail search "urgent"                    # Find messages
gt mail search "error" --from witness      # Filter by sender
gt mail search "status" --subject          # Search subjects only
gt mail search "" --archive                # Include archived
```

#### Queues and Claims

```bash
gt mail claim <queue-name>     # Claim from work queue
gt mail release <message-id>   # Release claimed message
```

### gt nudge

Send synchronous messages to any Gas Town worker.

```bash
gt nudge <target> <message>
gt nudge gastown/toast "Check your mail and start working"
gt nudge witness "Check polecat health"
gt nudge mayor "Status update requested"
gt nudge channel:workers "New priority work"
```

**Flags:**
- `-m, --message <text>` - Message to send
- `-f, --force` - Send even if target has DND enabled

**Role Shortcuts:**
- `mayor` -> gt-mayor session
- `deacon` -> gt-deacon session
- `witness` -> Current rig's witness
- `refinery` -> Current rig's refinery

**Channel Syntax:**
```bash
gt nudge channel:<name> "Message"
```
Channels defined in `~/gt/config/messaging.json`

**DND (Do Not Disturb):**
- Respects target's notification settings
- Use `--force` to override

### gt broadcast

Send nudge to all workers.

```bash
gt broadcast "Check your mail"
gt broadcast --rig gastown "New priority work available"
gt broadcast --all "System maintenance in 5 minutes"
gt broadcast --dry-run "Test message"
```

**Flags:**
- `--rig <name>` - Only broadcast to specific rig
- `--all` - Include infrastructure agents (mayor, witness, etc.)
- `--dry-run` - Show what would be sent

### gt handoff

Hand off to a fresh session with context preservation.

```bash
gt handoff                          # Hand off current session
gt handoff -c                       # Collect state into handoff mail
gt handoff -s "Context" -m "Notes"  # Custom handoff message
gt handoff gt-abc                   # Hook bead, then restart
gt handoff crew                     # Hand off crew session
gt handoff mayor                    # Hand off mayor session
```

**Flags:**
- `-s, --subject <text>` - Subject for handoff mail
- `-m, --message <text>` - Message body
- `-c, --collect` - Auto-collect state (inbox, hooks, ready beads)
- `-w, --watch` - Switch to new session
- `-n, --dry-run` - Preview what would happen

**Collected State (with -c):**
- Hooked work
- Inbox summary
- Ready beads
- In-progress items

---

## Services

### gt up

Bring up all Gas Town services.

```bash
gt up                      # Start infrastructure
gt up --restore            # Also start crew and polecats with hooks
gt up --quiet              # Only show errors
```

**Started Services:**
- Daemon - Go background process
- Deacon - Health orchestrator
- Mayor - Global coordinator
- Witnesses - Per-rig polecat managers
- Refineries - Per-rig merge processors

**With --restore:**
- Crew from settings (settings/config.json)
- Polecats with pinned beads

### gt down

Stop all Gas Town services.

```bash
gt down                    # Stop infrastructure
gt down --polecats         # Also stop all polecats
gt down --all              # Also stop bd daemons
gt down --nuke             # Kill tmux server (DESTRUCTIVE)
gt down --dry-run          # Preview shutdown
```

**Shutdown Levels:**
1. Default: Infrastructure only (refineries, witnesses, mayor, deacon, daemon)
2. `--polecats`: Also stop polecat sessions
3. `--all`: Also stop bd daemons and verify
4. `--nuke`: Kill entire tmux server

**Flags:**
- `-q, --quiet` - Only show errors
- `-f, --force` - Skip graceful shutdown
- `-p, --polecats` - Stop all polecat sessions
- `-a, --all` - Full shutdown with verification
- `--nuke` - Kill tmux server (requires `GT_NUKE_ACKNOWLEDGED=1`)
- `--dry-run` - Preview actions

### gt daemon

Control the Gas Town daemon process.

```bash
gt daemon run              # Start daemon
gt daemon status           # Check status
gt daemon stop             # Stop daemon
```

**Daemon Responsibilities:**
- Background Go process
- Periodic agent health pokes
- Event monitoring

### gt boot

Boot the system watchdog.

```bash
gt boot start              # Start boot watchdog
gt boot status             # Check status
gt boot stop               # Stop watchdog
```

### gt feed

View activity feed.

```bash
gt feed                    # Show recent activity
gt feed --follow           # Continuous stream
gt feed --type nudge       # Filter by event type
gt feed --json             # JSON output
```

**Event Types:**
- `nudge` - Agent notifications
- `handoff` - Session handoffs
- `boot` - Service starts
- `halt` - Service stops

---

## Workspace

### gt rig

Manage rigs (project containers) in the workspace.

#### Add Rig

```bash
gt rig add <name> <git-url>
gt rig add gastown https://github.com/user/gastown
gt rig add myproject git@github.com:user/repo.git --prefix mp
gt rig add myproject <url> --branch develop
```

**Flags:**
- `--prefix <prefix>` - Beads issue prefix (default: derived from name)
- `--local-repo <path>` - Local repo to share git objects
- `--branch <name>` - Default branch (default: auto-detected)

**Created Structure:**
```
<name>/
  config.json
  .repo.git/        (shared bare repo)
  .beads/           (rig-level issues)
  plugins/          (rig-level plugins)
  mayor/rig/        (Mayor's clone)
  refinery/rig/     (Refinery worktree)
  crew/             (empty)
  witness/
  polecats/
```

#### List Rigs

```bash
gt rig list                # List all rigs
```

#### Remove Rig

```bash
gt rig remove <name>       # Remove from registry (keeps files)
```

#### Rig Status

```bash
gt rig status              # Infer rig from cwd
gt rig status gastown      # Specific rig
```

**Shows:**
- Rig information (name, path, prefix)
- Witness status
- Refinery status and queue
- Polecats (name, state, issue)
- Crew (name, branch, git status)

#### Rig Lifecycle

```bash
gt rig boot <name>         # Start witness + refinery
gt rig start <name>...     # Start multiple rigs
gt rig shutdown <name>     # Graceful shutdown
gt rig stop <name>...      # Stop multiple rigs
gt rig reboot <name>       # Restart rig
gt rig restart <name>...   # Restart multiple rigs
```

**Safety Checks:**
- Checks for uncommitted polecat work
- Use `--force` to skip graceful shutdown
- Use `--nuclear` to bypass ALL checks (loses uncommitted work!)

#### Rig Reset

```bash
gt rig reset               # Reset all state
gt rig reset --handoff     # Clear handoff content
gt rig reset --mail        # Clear stale mail
gt rig reset --stale       # Reset orphaned in_progress issues
gt rig reset --dry-run     # Preview changes
```

### gt crew

Manage persistent crew workspaces for humans.

#### Add Crew

```bash
gt crew add <name>                    # Create workspace
gt crew add joe max emma              # Create multiple
gt crew add fred --rig gastown        # Specific rig
gt crew add alice --branch            # Create feature branch
```

#### List Crew

```bash
gt crew list                          # List in current rig
gt crew list --rig gastown            # List in specific rig
gt crew list --all                    # List in all rigs
gt crew list --json                   # JSON output
```

#### Manage Sessions

```bash
gt crew start <rig> [names...]        # Start crew sessions
gt crew start beads                   # Start all crew in beads
gt crew start beads grip fang         # Start specific crew

gt crew stop [names...]               # Stop sessions
gt crew stop beads                    # Stop all in beads
gt crew stop --all                    # Stop all running

gt crew restart [names...]            # Restart sessions
gt crew restart --all                 # Restart all running
gt crew restart --dry-run             # Preview
```

#### Attach to Crew

```bash
gt crew at <name>                     # Attach to session
gt crew at                            # Auto-detect from cwd
gt crew at dave --detached            # Start without attaching
gt crew at dave --no-tmux             # Just print path
gt crew at dave --agent gemini        # Use different agent
```

**Flags:**
- `--rig <name>` - Specify rig
- `--no-tmux` - Print directory only
- `-d, --detached` - Start without attaching
- `--account <handle>` - Claude Code account
- `--agent <alias>` - Agent override

#### Other Commands

```bash
gt crew remove <name...>              # Remove workspace(s)
gt crew remove dave --force           # Skip safety checks
gt crew remove test --purge           # Delete agent bead entirely

gt crew refresh <name>                # Context cycle with handoff
gt crew refresh dave -m "Notes"       # Custom message

gt crew status [name]                 # Detailed status
gt crew rename <old> <new>            # Rename workspace
gt crew pristine [name]               # Sync with remote
```

### gt init

Initialize a new Gas Town workspace.

```bash
gt init                    # Initialize current directory
gt init <path>             # Initialize specific path
gt init --name "My Town"   # Set town name
```

### gt install

Install Gas Town dependencies and configuration.

```bash
gt install                 # Full installation
gt install --check         # Verify installation
gt install --hooks         # Install git hooks only
```

---

## Configuration

### gt config

Manage Gas Town configuration.

#### Agent Configuration

```bash
gt config agent list                    # List all agents
gt config agent list --json             # JSON output
gt config agent get <name>              # Show agent config
gt config agent set <name> <command>    # Set custom agent
gt config agent remove <name>           # Remove custom agent
```

**Built-in Agents:**
- `claude` - Claude Code (default)
- `gemini` - Gemini CLI
- `codex` - Codex CLI

**Custom Agent Example:**
```bash
gt config agent set claude-glm "claude-glm --model glm-4"
```

#### Default Agent

```bash
gt config default-agent                 # Show current
gt config default-agent claude          # Set default
```

#### Agent Email Domain

```bash
gt config agent-email-domain                    # Show current
gt config agent-email-domain gastown.local      # Set domain
```

Used for git commit emails: `gastown/crew/joe` -> `gastown.crew.joe@domain`

### gt role

Display current role information.

```bash
gt role                    # Show detected role
gt role --json             # JSON output
```

### gt plugin

Manage plugins.

```bash
gt plugin list             # List installed plugins
gt plugin install <name>   # Install plugin
gt plugin remove <name>    # Remove plugin
```

### gt theme

Manage visual themes.

```bash
gt theme list              # List themes
gt theme set <name>        # Set active theme
gt theme show              # Show current theme
```

---

## Diagnostics

### gt doctor

Run health checks on the workspace.

```bash
gt doctor                  # Run all checks
gt doctor --fix            # Auto-fix issues
gt doctor --rig gastown    # Check specific rig
gt doctor --verbose        # Detailed output
```

**Check Categories:**

**Workspace:**
- `town-config-exists` - Check mayor/town.json
- `town-config-valid` - Validate config
- `rigs-registry-exists` - Check rigs.json (fixable)
- `mayor-exists` - Check mayor/ structure

**Town Root Protection:**
- `town-git` - Verify version control
- `town-root-branch` - Verify on main branch (fixable)
- `pre-checkout-hook` - Verify hook exists (fixable)

**Infrastructure:**
- `stale-binary` - Check gt binary freshness
- `daemon` - Check daemon running (fixable)
- `repo-fingerprint` - Validate fingerprint (fixable)
- `boot-health` - Check watchdog health

**Cleanup (fixable):**
- `orphan-sessions` - Detect orphaned tmux sessions
- `orphan-processes` - Detect orphaned Claude processes
- `wisp-gc` - Clean abandoned wisps (>1h)

**Clone Divergence:**
- `persistent-role-branches` - Detect non-main branches
- `clone-divergence` - Detect behind origin/main

**Rig Checks (with --rig):**
- `rig-is-git-repo` - Valid git repository
- `git-exclude-configured` - Check .git/info/exclude (fixable)
- `witness-exists` - Verify witness structure (fixable)
- `refinery-exists` - Verify refinery structure (fixable)
- `polecat-clones-valid` - Verify polecat directories

### gt status

Show overall town status.

```bash
gt status                  # Show status
gt status --json           # JSON output
gt status --fast           # Skip mail lookups
gt status --watch          # Continuous refresh
gt status -n 5             # 5-second interval
gt status --verbose        # Detailed per-agent
```

**Displays:**
- Town name and location
- Overseer (human operator) info
- Global agents (Mayor, Deacon)
- Per-rig status (witness, refinery, crew, polecats)
- Merge queue summary
- Hook status and mail counts

### gt log

View Gas Town logs.

```bash
gt log                     # Show recent logs
gt log --follow            # Tail logs
gt log --rig gastown       # Filter by rig
gt log --level error       # Filter by level
```

### gt costs

View API usage costs.

```bash
gt costs                   # Show cost summary
gt costs --period day      # Daily breakdown
gt costs --period week     # Weekly breakdown
gt costs --json            # JSON output
```

### gt activity

View activity history.

```bash
gt activity                # Recent activity
gt activity --rig gastown  # Filter by rig
gt activity --agent toast  # Filter by agent
gt activity --json         # JSON output
```

---

## Typical Workflows

### Starting Work for the Day

```bash
# Start all services
gt up

# Check status
gt status

# Attach to crew workspace
gt crew at dave
```

### Assigning Work

```bash
# Create a task
bd create "Fix login bug" --type task --priority 1

# Assign to a polecat
gt sling gt-abc123 gastown/toast --nudge

# Or let Witness auto-assign
# (Witness monitors ready queue and spawns polecats)
```

### Ending a Session

```bash
# Save context and hand off
gt handoff -c

# Or complete work and signal done
gt done --exit COMPLETE
```

### Stopping for the Night

```bash
# Stop all services
gt down

# Or stop polecats but keep infrastructure
gt down --polecats
```

### Debugging Issues

```bash
# Run health checks
gt doctor

# Auto-fix common issues
gt doctor --fix

# Check specific rig
gt doctor --rig gastown --verbose
```

### Communication Between Agents

```bash
# Send message
gt mail send gastown/witness -s "Status" -m "How are the polecats?"

# Check inbox
gt mail inbox

# Read and reply
gt mail read msg-abc123
gt mail reply msg-abc123 -m "All good!"
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `GT_ROLE` | Current agent role |
| `GT_RIG` | Current rig name |
| `GT_POLECAT` | Polecat name (if applicable) |
| `GT_CREW` | Crew name (if applicable) |
| `GT_AGENT` | Agent override |
| `GT_SESSION_ID` | Session identifier |
| `BD_ACTOR` | Actor identity for beads |
| `TMUX_PANE` | Current tmux pane |
| `ANTHROPIC_API_KEY` | Claude API key |
| `GT_NUKE_ACKNOWLEDGED` | Required for destructive operations |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Not in workspace |
| 4 | Agent not found |
| 5 | Permission denied |

---

## See Also

- [Agent Types](02-agent-types.md) - Detailed agent documentation
- [Data Models](04-data-models.md) - Beads and data structures
- [Workflows](05-workflows.md) - Common workflow patterns
- [GTN TUI](06-gtn-tui.md) - Terminal user interface
