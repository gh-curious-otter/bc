# bc - An AI Agent Orchestration System for Software Development

`bc` is a powerful, CLI-first agent orchestration system designed to coordinate multiple AI agents for software development tasks. It provides a structured, predictable, and cost-aware environment for AI-driven software engineering.

## Core Philosophy

- **Organic Growth:** Start with a single root agent and expand your team conversationally. No rigid, predefined structures are enforced.
- **CLI-First:** Every feature is accessible through the `bc` command-line interface. The Terminal User Interface (TUI) is a powerful tool for navigation and visualization, similar to `k9s` for Kubernetes.
- **Agent Agnostic:** `bc` supports a variety of AI agents, including Claude Code, Cursor Agent, and OpenAI Codex. You can also configure custom agents.
- **Persistent Memory:** Agents learn and retain information across sessions, building a knowledge base over time to become more effective.

## Features

- **Hierarchical Agent System:** A root agent manages the main branch and can spawn a team of agents with different roles (e.g., manager, engineer, QA).
- **Isolated Workspaces:** Each agent operates in its own Git worktree, ensuring conflict-free parallel development.
- **Real-Time Communication:** Agents collaborate through a Slack-like channel system, with support for mentions, and message history.
- **TUI Dashboard:** A comprehensive TUI (`bc home`) provides a real-time overview of your agents, processes, and project status.
- **Persistent Memory:** Agents have a persistent memory system, allowing them to learn from past experiences and improve their performance.
- **Scheduled Tasks (Demons):** Automate recurring tasks like daily builds, nightly tests, or weekly dependency updates.
- **Process Management:** Start, monitor, and manage long-running processes like development servers or simulators.
- **Unified Issue Tracking:** A consolidated view of issues from different sources (e.g., Beads, GitHub).
- **Hierarchical Merge Queues:** A structured workflow for code review and integration, preventing conflicts from reaching the main branch.

## Installation

### Prerequisites

- Go 1.25.1+
- tmux
- An AI agent (e.g., Claude Code, Cursor Agent)

### Build from Source

```bash
# Build the bc executable
make build

# Install to your GOPATH/bin
make install
```

## Getting Started

1.  **Initialize a Workspace:**
    ```bash
    bc init
    ```

2.  **Start the Root Agent:**
    ```bash
    bc up
    ```

3.  **Open the TUI Dashboard:**
    ```bash
    bc home
    ```

4.  **Spawn a New Agent:**
    ```bash
    bc spawn my-engineer --role engineer
    ```

5.  **Send a Message to an Agent:**
    ```bash
    bc send my-engineer "Please implement the login feature."
    ```

6.  **Stop All Agents:**
    ```bash
    bc down
    ```

## CLI Command Reference

| Command | Description |
| :--- | :--- |
| **Workspace** | |
| `bc init` | Initialize a new `bc` workspace. |
| `bc home` | Open the TUI dashboard. |
| `bc status` | Show a summary of the workspace status. |
| `bc up` | Start the root agent. |
| `bc down` | Stop all agents and processes. |
| **Agents** | |
| `bc agent create <name> --role <role>` | Create a new agent. |
| `bc agent list` | List all agents in the workspace. |
| `bc agent peek <name>` | Get a live view of an agent's activity. |
| `bc agent attach <name>` | Attach to an agent's interactive session. |
| `bc agent send <name> <message>` | Send a message to an agent. |
| `bc agent stop <name>` | Stop a specific agent. |
| **Channels** | |
| `bc channel list` | List all channels. |
| `bc channel create <name>` | Create a new channel. |
| `bc channel send <channel> <message>` | Send a message to a channel. |
| `bc channel history <channel>` | View the message history of a channel. |
| **Memory** | |
| `bc memory show <agent>` | View an agent's memory summary. |
| `bc memory search <agent> <query>` | Search an agent's experiences. |
| `bc memory clear <agent>` | Reset an agent's memory. |
| **And many more...** | |

For a full list of commands and their options, use `bc help` and `bc <command> --help`.

## Documentation

For more in-depth information, please refer to the documents in the `.ctx` directory:

-   [Architecture Overview](.ctx/01-architecture-overview.md)
-   [Agent Roles](.ctx/02-agent-types.md)
-   [CLI Reference](.ctx/03-cli-reference.md)
-   [Data Models](.ctx/04-data-models.md)
-   [Workflows](.ctx/05-workflows.md)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for more details.
