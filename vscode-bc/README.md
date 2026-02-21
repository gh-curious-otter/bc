# BC Agent Orchestration - VS Code Extension

VS Code integration for the [bc](https://github.com/rpuneet/bc) AI agent orchestration CLI.

## Features

### Sidebar Views
- **Agents View**: See all agents with status, role, and current task
- **Channels View**: Browse communication channels
- **Processes View**: Monitor background processes

### Commands (Command Palette)
- `BC: Show Status` - Display workspace status
- `BC: List Agents` - Refresh agent list
- `BC: Create Agent` - Create a new agent
- `BC: Start Agent` - Start a stopped agent
- `BC: Stop Agent` - Stop a running agent
- `BC: Attach to Agent` - Open terminal attached to agent session
- `BC: Peek Agent Output` - View recent agent output
- `BC: Send Channel Message` - Send message to a channel
- `BC: View Channel History` - View channel messages
- `BC: Refresh` - Refresh all views

### Status Bar
Shows active/total agent count in the status bar.

## Requirements

- VS Code 1.85.0 or higher
- `bc` CLI installed and in PATH

## Installation

### From VSIX (Local)
1. Download the `.vsix` file
2. In VS Code: `Extensions` → `...` → `Install from VSIX...`
3. Select the downloaded file

### From Source
```bash
cd vscode-bc
npm install
npm run compile
npm run package
# Install the generated .vsix file
```

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `bc.binaryPath` | `bc` | Path to bc binary |
| `bc.refreshInterval` | `5000` | Auto-refresh interval (ms) |
| `bc.showStatusBar` | `true` | Show status bar item |

## Usage

1. Open a folder containing a `.bc` directory
2. The extension activates automatically
3. Use the BC sidebar or Command Palette commands

### Quick Actions
- Click an agent to see details
- Right-click agents for context actions (start/stop/peek)
- Click a channel to view history
- Status bar shows agent count

## Development

```bash
# Install dependencies
npm install

# Compile
npm run compile

# Watch mode
npm run watch

# Lint
npm run lint

# Package
npm run package
```

## License

MIT
