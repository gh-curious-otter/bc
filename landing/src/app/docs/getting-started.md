# Getting Started with bc

Welcome to **bc** – the multi-agent orchestration system for coordinated software development. This guide will walk you through installation, setup, and your first workflow.

---

## Table of Contents

1. [Installation](#installation)
2. [Initial Setup](#initial-setup)
3. [Your First Workflow](#your-first-workflow)
4. [Common Commands](#common-commands)
5. [Troubleshooting](#troubleshooting)
6. [Next Steps](#next-steps)

---

## Installation

### macOS

**Using Homebrew:**
```bash
brew install bc-infra/bc/bc
bc --version
```

**From Source:**
```bash
git clone https://github.com/bcinfra1/bc.git
cd bc
cargo install --path .
bc --version
```

### Linux

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install bc-cli
bc --version
```

**From Source:**
```bash
git clone https://github.com/bcinfra1/bc.git
cd bc
cargo install --path .
bc --version
```

### Windows

**Using Chocolatey:**
```bash
choco install bc-cli
bc --version
```

**Manual Installation:**
1. Download latest release from [bc-infra/releases](https://github.com/bcinfra1/bc/releases)
2. Extract to `C:\Program Files\bc`
3. Add to PATH: `C:\Program Files\bc\bin`
4. Verify: Open PowerShell and run `bc --version`

---

## Initial Setup

### 1. Initialize Your Workspace

```bash
# Create a new project directory
mkdir my-project && cd my-project

# Initialize bc workspace
bc init

# Verify setup
bc status
```

**Output:**
```
✓ Workspace initialized (.bc/)
✓ Config file created (.bc/config.yaml)
✓ Ready to create agents
```

### 2. Start the Root Agent

```bash
# Start root coordinator
bc up

# Check status
bc status
```

**Expected Output:**
```
ROOT AGENT: running
  • State: idle
  • Uptime: 0h 2m
```

### 3. Create Your First Agent

```bash
# Create a manager agent
bc agent create manager-atlas --role manager --tool cursor

# Create an engineer agent
bc agent create engineer-pixel --role engineer --tool claude

# List agents
bc agent list
```

**Output:**
```
AGENTS (3):
  • root-prime (root, 100% uptime)
  • manager-atlas (manager, idle)
  • engineer-pixel (engineer, idle)
```

---

## Your First Workflow

### Scenario: Build a Feature with bc

**Step 1: Assign Work via Channels**

```bash
# Send task assignment to #engineering channel
bc channel send engineering "@engineer-pixel: Take task #1 - user auth implementation. Use jwt tokens, verify against db. DM when ready."

# Check messages
bc channel history engineering --limit 5
```

**Step 2: Engineer Works on Task**

```bash
# Simulate engineer picking up work
bc agent peek engineer-pixel
```

**Output:**
```
AGENT: engineer-pixel (state: tool)
  ⏺ created branch: feat/user-auth
  ⏺ implementing jwt middleware
  ✽ pushing to remote
  ✻ running tests
```

**Step 3: Review & Announce**

```bash
# Post status to channel
bc channel send general "🎉 Task complete! User auth feature merged and live. Great work team!"
```

---

## Common Commands

### Agent Management

```bash
# View all agents
bc agent list

# Peek at agent's current work
bc agent peek engineer-pixel

# Send direct message to agent
bc agent send engineer-pixel "Status update please?"

# Attach to agent's live session
bc agent attach engineer-pixel
```

### Cost Tracking

```bash
# View cost summary
bc cost show

# View token usage
bc cost usage

# Set budget for an agent
bc cost budget set 50.00 --agent engineer-pixel
```

### Channels (Team Communication)

```bash
# List channels
bc channel list

# Send message to channel
bc channel send #general "Update: Feature X shipped 🚀"

# Check channel history
bc channel history #engineering --limit 10

# Create new channel
bc channel create product-team
```

### Scheduled Tasks (Cron)

```bash
# List scheduled tasks
bc cron list

# Run a cron job manually
bc cron run nightly-tests

# Create new cron job
bc cron add test-suite --schedule "0 2 * * *" --command "make test"

# View cron logs
bc cron logs test-suite
```

---

## Troubleshooting

### Issue: "Workspace not initialized"

**Solution:**
```bash
# Make sure you're in project directory
pwd

# If needed, reinitialize
bc init

# Verify
bc status
```

### Issue: Agent stuck or unresponsive

**Solution:**
```bash
# Check agent status
bc agent list

# Restart agent (stop then start)
bc agent stop engineer-pixel
bc agent start engineer-pixel

# View agent logs
bc agent logs engineer-pixel --since 1h
```

### Issue: Merge conflict preventing PR merge

**Solution:**
```bash
# Check agent status to see if it's stuck
bc status

# Send the agent instructions to resolve conflicts
bc agent send engineer-pixel "Resolve merge conflicts on your branch and push"

# Monitor progress
bc agent peek engineer-pixel
```

### Issue: Channel messages not appearing

**Solution:**
```bash
# Verify channel exists
bc channel list

# Verify you're sending to correct channel
bc channel history #general

# Resend message
bc channel send #general "Test message"
```

### Issue: Performance slow or timeouts

**Solution:**
```bash
# Check system resources
bc doctor

# Run auto-fix for common issues
bc doctor fix

# Restart agents
bc down && bc up
```

---

## Next Steps

### 1. Explore Documentation
- [API Reference](/docs/api)
- [CLI Reference](/docs/cli)
- [Architecture Guide](/docs/architecture)

### 2. Build Your Team Structure
- Create agents for your team roles (engineers, QA, product)
- Set up channels for team communication
- Define workflows and automation

### 3. Integrate with Tools
- Connect GitHub for PR management
- Link Jira for task tracking
- Setup Slack notifications

### 4. Advanced Features
- Custom agent behaviors
- Performance optimization
- Audit logging and compliance

### 5. Get Support
- GitHub Issues: [Report bugs](https://github.com/bcinfra1/bc/issues)
- Documentation: [Full docs](https://docs.bc-infra.com)
- Community: [Discord server](https://discord.gg/bc-infra)

---

## Example: Complete Workflow

Here's a realistic end-to-end workflow:

```bash
# 1. Initialize
bc init && bc up

# 2. Create team
bc agent create pm-alex --role manager --tool cursor
bc agent create eng-sam --role engineer --tool claude
bc agent create qa-jamie --role engineer --tool gemini

# 3. Assign work via channel
bc channel send engineering "@eng-sam: Build user profile page. UI design in #product. Ship within 2 hours?"

# 4. Monitor progress
bc agent peek eng-sam
bc channel history engineering

# 5. Check costs
bc cost show

# 6. Announce
bc channel send general "User profile feature live! Thanks team!"
```

---

## Quick Reference Card

| Task | Command |
|------|---------|
| Check status | `bc status` |
| List agents | `bc agent list` |
| Send message | `bc channel send engineering "message"` |
| View agent work | `bc agent peek engineer-name` |
| Schedule task | `bc cron add name --schedule "0 2 * * *" --command "make test"` |
| View costs | `bc cost show` |
| View logs | `bc agent logs agent-name --since 1h` |
| Health check | `bc doctor` |
| Get help | `bc --help` |

---

## Getting Help

**Command line help:**
```bash
bc --help
bc <command> --help
```

**Documentation:**
- [Full Documentation](https://docs.bc-infra.com)
- [API Docs](https://docs.bc-infra.com/api)
- [CLI Reference](https://docs.bc-infra.com/cli)

**Support:**
- [GitHub Issues](https://github.com/bcinfra1/bc/issues)
- [Discord Community](https://discord.gg/bc-infra)
- Email: support@bc-infra.com

---

**Happy building! 🚀**

*Last Updated: 2026-02-09*
