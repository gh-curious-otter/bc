# bc VS Code Extension

VS Code extension for bc (AI Agent Orchestration CLI).

## Features

- **Agent Status Panel** - View all agents and their states in sidebar
- **Channel Panel** - Browse and send messages to channels
- **Status Bar Widget** - Quick view of workspace status (agent count, working)
- **Command Palette** - Access all bc commands via `Ctrl+Shift+P`
- **Keyboard Shortcuts** - Quick status (`Ctrl+Alt+B` / `Cmd+Alt+B`)
- **Auto-detection** - Extension activates in bc workspaces

## Requirements

- VS Code 1.85+
- bc CLI installed and in PATH
- Project must be a bc workspace (contains `.bc` directory)

## Installation

### From VS Code Marketplace (Recommended)

1. Open Extensions (`Ctrl+Shift+X`)
2. Search for "bc"
3. Click Install

### Manual Installation

1. Build the extension: `npm run compile`
2. Package: `npx vsce package`
3. Install from VSIX: Extensions → Install from VSIX

## Usage

### Sidebar

Open the bc sidebar from the Activity Bar (robot icon):
- **Agents** - Shows agent list with status indicators
- **Channels** - Shows available channels

### Commands

Access from Command Palette (`Ctrl+Shift+P`):
- **bc: Show Status** - Display workspace status
- **bc: List Agents** - Show all agents in quick pick
- **bc: Agent Health** - Check agent health
- **bc: List Channels** - Show available channels
- **bc: Send to Channel** - Send message to channel
- **bc: View Logs** - Show recent bc logs
- **bc: Refresh** - Refresh all views

### Keyboard Shortcuts

- `Ctrl+Alt+B` / `Cmd+Alt+B` - Quick status popup

### Settings

Configure at Settings → Extensions → bc:
- **bc.binaryPath** - Path to bc binary (default: `bc`)
- **bc.refreshInterval** - Auto-refresh interval in seconds (default: 30, 0 to disable)

## Development

### Building

```bash
npm install
npm run compile
```

### Testing

```bash
npm test
```

### Running in Development

1. Open this folder in VS Code
2. Press F5 to launch Extension Development Host
3. Open a bc workspace folder

### Publishing

```bash
npx vsce publish
```

## License

MIT
