# bc Demo Walkthrough

A step-by-step guide for demonstrating bc multi-agent orchestration in 5-10 minutes.

---

## Prerequisites

Before starting the demo:

- [ ] bc installed and in PATH (`bc --version`)
- [ ] Terminal with tmux support
- [ ] GitHub CLI authenticated (`gh auth status`)
- [ ] Clean terminal window (larger font for visibility)

## Demo Overview

This demo showcases a complete bug fix workflow:
1. PM identifies a bug
2. Manager assigns to engineer
3. Engineer fixes in isolated worktree
4. QA validates the fix
5. Manager merges to main

---

## Demo Script (5-10 minutes)

### Step 1: Initialize Workspace (30 sec)

**What to say:** "Let's start with a fresh project. bc uses a workspace model where all agent state lives in a .bc directory."

```bash
# Navigate to demo project
cd examples/demo-project

# Initialize bc workspace (if not already done)
bc init

# Show workspace structure
ls -la .bc/
```

**Expected output:**
```
.bc/
├── agents.json      # Agent state
├── queue.json       # Work queue
├── channels.json    # Communication channels
└── worktrees/       # Per-agent git worktrees
```

**Key point:** "Each agent gets its own git worktree - complete isolation."

---

### Step 2: Start the Agent Team (1 min)

**What to say:** "Now we spin up our AI agent team. bc uses a hierarchy: PM, Manager, Engineers, and QA."

```bash
# Start the full team
bc up

# Watch agents spawn (wait 5-10 seconds)
bc status
```

**Expected output:**
```
AGENT           ROLE            STATE     TASK
────────────────────────────────────────────────
coordinator     coordinator     working   Initializing...
product-manager product-manager idle
manager         manager         idle
engineer-01     engineer        idle
engineer-02     engineer        idle
qa-01           qa              idle
```

**Key point:** "All agents are running in tmux sessions. They have different roles and capabilities."

---

### Step 3: Create a Bug Report (1 min)

**What to say:** "Let's report a bug. The greeting module has a typo - it says 'Helo' instead of 'Hello'."

```bash
# Show the bug
cat src/greeting.go

# PM creates an issue via channel broadcast
bc send product-manager "There's a typo bug in greeting.go - it says 'Helo' instead of 'Hello'. Please create a work item to fix it."
```

**What happens:** PM analyzes and creates a work item in the queue.

```bash
# After ~30 seconds, check the queue
bc queue
```

**Expected output:**
```
ID        STATUS    ASSIGNED  TITLE
──────────────────────────────────────────
work-001  pending   -         Fix greeting typo: Helo → Hello
```

**Key point:** "The PM understood the bug and created a structured work item."

---

### Step 4: Manager Assigns Work (1 min)

**What to say:** "The manager sees pending work and assigns it to an available engineer."

```bash
# Manager assigns the work
bc send manager "Please assign work-001 to an available engineer."

# Wait for assignment, then check queue
bc queue
```

**Expected output:**
```
ID        STATUS    ASSIGNED      TITLE
──────────────────────────────────────────────
work-001  working   engineer-01   Fix greeting typo: Helo → Hello
```

**Key point:** "Work is now assigned. The engineer will work in their own isolated worktree."

---

### Step 5: Engineer Fixes the Bug (2-3 min)

**What to say:** "Let's watch the engineer work. They get their own git branch and worktree."

```bash
# Attach to engineer's session to watch
bc attach engineer-01
```

**What you'll see:** The engineer:
1. Creates a branch: `engineer-01/work-001/fix-greeting-typo`
2. Edits `src/greeting.go` to fix "Helo" → "Hello"
3. Runs tests: `go test ./...`
4. Commits: `git commit -m "fix: correct greeting typo"`
5. Reports done: `bc report done "Fixed greeting typo"`

**To exit:** Press `Ctrl+B` then `D` to detach from tmux.

```bash
# Check status after fix
bc status
bc queue
```

**Expected output:**
```
work-001  done   engineer-01   Fix greeting typo: Helo → Hello
```

**Key point:** "The engineer worked in isolation. Main branch is untouched until merge."

---

### Step 6: QA Validates (1-2 min)

**What to say:** "Before merging, QA validates the fix."

```bash
# Send to QA for validation
bc send qa-01 "Please validate the fix in work-001 - verify the greeting typo is corrected."

# Optionally attach to watch
bc attach qa-01
```

**What QA does:**
1. Checks out the engineer's branch
2. Runs tests
3. Verifies the fix: `grep "Hello" src/greeting.go`
4. Approves: `bc report done "QA validated - fix is correct"`

**Key point:** "QA is independent - they verify before anything goes to main."

---

### Step 7: Manager Merges (1 min)

**What to say:** "With QA approval, the manager merges to main."

```bash
# Manager handles the merge
bc send manager "Work-001 is validated by QA. Please merge to main."

# Watch the merge happen
bc attach manager
```

**What happens:**
1. Manager merges the branch to main
2. Updates work item status
3. Cleans up the feature branch

```bash
# Verify on main
git log --oneline -3
```

**Expected output:**
```
abc1234 fix: correct greeting typo
def5678 Previous commit...
```

**Key point:** "Clean merge to main with full traceability."

---

### Step 8: Wrap Up (30 sec)

**What to say:** "Let's see the final state."

```bash
# Show all agents idle, work complete
bc status

# Show queue history
bc queue

# Show the fixed code
cat src/greeting.go
```

**Final message:** "In under 10 minutes, we went from bug report to merged fix - with AI agents handling the workflow, isolation, and coordination."

```bash
# Clean up when done
bc down
```

---

## Key Points to Highlight

Throughout the demo, emphasize these differentiators:

### 1. Agent Hierarchy
- PM creates high-level work items
- Manager breaks down and assigns
- Engineers implement
- QA validates

### 2. Worktree Isolation
- Each agent has their own git worktree
- No conflicts between concurrent work
- Main branch stays clean until merge

### 3. Real-Time Visibility
- `bc status` shows all agent states
- `bc attach` lets you watch any agent
- `bc queue` tracks all work items

### 4. Structured Communication
- Channels for broadcast messages
- Direct sends to specific agents
- Clear handoffs between roles

---

## Troubleshooting

### Agents not starting
```bash
# Check if tmux is running
tmux list-sessions

# Kill and restart
bc down
bc up
```

### Agent stuck
```bash
# Check agent state
bc status

# Send a nudge
bc send <agent> "Please continue with your current task."
```

### Work item not appearing
```bash
# Force queue refresh
bc queue

# Check if beads sync is needed
bc queue load
```

### Can't attach to agent
```bash
# List tmux sessions directly
tmux list-sessions

# Attach manually
tmux attach -t bc-<agent-name>
```

---

## Demo Variations

### Quick Demo (3 min)
- Skip Steps 6-7 (QA and merge)
- Focus on: init → start → create bug → assign → fix

### Extended Demo (15 min)
- Add a second bug fix in parallel
- Show multiple engineers working simultaneously
- Demonstrate channel broadcasts for team updates

### Technical Deep Dive
- Show `.bc/` directory contents
- Explain worktree structure
- Demonstrate `bc attach` with live coding

---

## Post-Demo Questions

**Q: How do agents communicate?**
A: Via tmux send-keys. bc sends messages directly to agent sessions.

**Q: What AI models power the agents?**
A: Currently Claude Code, but the architecture supports any AI coding tool (Cursor, Codex, etc.).

**Q: Can I customize agent roles?**
A: Yes - role prompts are in `prompts/` directory. Each role has capabilities defined in code.

**Q: What happens if an agent crashes?**
A: bc tracks state in `.bc/agents.json`. Agents can be restarted and resume work.

**Q: Is this production-ready?**
A: v1 is functional but has known reliability gaps. v2 redesign is in planning.

---

*This walkthrough is part of the bc demo project. See README.md for project overview.*
