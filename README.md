# bc - AI Agent Orchestration for Software Development

`bc` is a CLI-first orchestration system for coordinating teams of AI agents to work on software development projects. It provides a structured, observable, and persistent environment for AI-driven engineering, emphasizing developer control and predictable behavior.

Drawing inspiration from `k9s` for Kubernetes, `bc` treats the Terminal User Interface (TUI) as a powerful visualization and navigation layer on top of a robust set of command-line tools.

## Core Philosophy

*   **Organic Growth**: Start with a single `root` agent and conversationally grow your team. The system is designed to be flexible, without enforcing a rigid, predefined structure.
*   **CLI-First**: Every feature is accessible and scriptable through the `bc` command line. The `bc home` TUI provides a real-time dashboard for observation and interaction.
*   **Agent Agnostic**: `bc` is designed to work with any AI agent that can run in a terminal. It has built-in support for Claude Code, Cursor, and OpenAI Codex, with easy configuration for custom tools.
*   **Persistent Memory**: Agents learn from their experiences. `bc` includes a memory system that allows agents to record task outcomes and accumulate knowledge, improving their effectiveness over time.
*   **Isolated Workspaces**: To prevent conflicts and enable parallel development, each agent operates within its own dedicated `git worktree`.

## Features

-   **Hierarchical Agent System**: A `root` agent manages the project and can create a team of agents with various roles (`engineer`, `qa`, `manager`, etc.). Roles are customizable via simple Markdown files.
-   **TUI Dashboard (`bc home`)**: An interactive, real-time dashboard for visualizing workspaces, agent status, communication channels, work queues, and running processes.
-   **Real-Time Communication (`bc channel`)**: Agents collaborate through Slack-like channels, with support for message history and agent-to-agent communication.
-   **Persistent Memory (`bc memory`)**: Agents can record experiences and learnings, search their memory, and improve over time.
-   **Scheduled Tasks (`bc demon`)**: Automate recurring tasks like daily builds, nightly tests, or health checks using cron-based scheduling.
-   **Process Management (`bc process`)**: Start, stop, and monitor long-running background processes like development servers, simulators, or builds.
-   **Git Worktree Management (`bc worktree`)**: Each agent gets an isolated `git worktree` for conflict-free development. `bc` provides tools to manage and prune these worktrees.
-   **Cost Tracking (`bc cost`)**: Track API costs with commands to view detailed records and summaries per-agent, per-team, or for the entire workspace.

## Installation

### Prerequisites

-   Go 1.25.1+
-   `tmux`
-   A configured AI agent tool (e.g., Claude Code, Cursor).

### Build from Source

```bash
# Build the bc executable
make build

# Install to your GOPATH/bin
make install
```

## Getting Started

1.  **Initialize a Workspace**:
    This creates a `.bc` directory in your project to store configuration, roles, and agent state.
    ```bash
    bc init
    ```

2.  **Start the Root Agent**:
    The `root` agent is the primary orchestrator for the workspace.
    ```bash
    bc up
    ```
    *Note: `bc up` can also be used to start a default team of agents, configurable in `.bc/config.toml`.*

3.  **Open the TUI Dashboard**:
    Use the TUI to get a real-time overview of your workspace.
    ```bash
    bc home
    ```

4.  **Create a New Agent**:
    Spawn a new agent with a specific role. `bc` will assign a random memorable name if one isn't provided.
    ```bash
    bc agent create --role engineer
    ```

5.  **Communicate with your Agent**:
    Send instructions to an agent or a channel.
    ```bash
    # Send a direct message to an agent
    bc agent send swift-falcon "Please implement the login feature."

    # Send a message to the 'engineering' channel
    bc channel send engineering "Starting work on the new auth system."
    ```

6.  **Stop the System**:
    Gracefully shut down all running agents and their sessions.
    ```bash

    bc down
    ```

## CLI Command Reference

`bc` provides a rich set of commands for managing the entire agent ecosystem. Below is a summary of the main commands. For detailed options, run `bc [command] --help`.

| Command | Description |
| :--- | :--- |
| **Workspace & Lifecycle** | |
| `init` | Initialize a new `bc` workspace in the current directory. |
| `up` | Start the `root` agent and the configured agent roster. |
| `down` | Stop all running agents and processes. |
| `home` | Open the interactive TUI dashboard. |
| `status` | Show the status of all agents. |
| `dashboard`| Show a summary of workspace stats. |
| **Agent Management** | |
| `agent list`| List all agents with their status, role, and current task. |
| `agent create`| Create and start a new agent with a specified role. |
| `agent send`| Send a message or command to an agent. |
| `agent attach`| Attach to an agent's `tmux` session for direct interaction. |
| `agent peek`| View recent output from an agent's session. |
| `agent stop`| Stop a specific agent. |
| **Collaboration & Workflow** | |
| `channel` | Manage communication channels for agents. |
| `team` | Organize agents into teams. |
| `queue` | Manage the work queue (integrates with GitHub Issues). |
| `merge` | Merge an agent's work branch into main after validation checks. |
| `role` | Manage custom agent roles and their prompt templates. |
| **Automation & Tooling** | |
| `demon` | Manage scheduled background tasks (demons) using cron syntax. |
| `process` | Manage long-running background processes (e.g., dev servers). |
| `memory` | Manage the persistent memory for each agent. |
| `worktree`| Manage agent-specific `git worktrees`. |
| `cost` | View and summarize API cost information. |
| **Configuration & System** | |
| `config` | Manage the workspace configuration (`.bc/config.toml`). |
| `logs` | View the `bc` event log with filtering capabilities. |
| `version` | Print the current version information. |


## Contributing

Contributions are welcome! Please see the [Contributing Guide](CONTRIBUTING.md) for more details on how to get involved.
