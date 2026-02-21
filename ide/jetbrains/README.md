# bc JetBrains Plugin

JetBrains IDE plugin for bc (AI Agent Orchestration CLI).

## Features

- **Agent Status Panel** - View all agents and their states in real-time
- **Channel Messages Panel** - Send and receive inter-agent messages
- **Status Bar Widget** - Quick view of workspace status
- **bc Commands Menu** - Access all bc commands from Tools menu
- **Keyboard Shortcuts** - Quick access (Ctrl+Alt+B for status)
- **Auto-detection** - Plugin activates in bc workspaces

## Requirements

- JetBrains IDE 2023.3+
- bc CLI installed and in PATH
- Project must be a bc workspace (contains `.bc` directory)

## Installation

### From JetBrains Marketplace (Recommended)

1. Open IDE Settings → Plugins
2. Search for "bc"
3. Click Install

### Manual Installation

1. Build the plugin: `./gradlew buildPlugin`
2. Open IDE Settings → Plugins → Install Plugin from Disk
3. Select `build/distributions/bc-jetbrains-*.zip`

## Usage

### Tool Windows

Open tool windows from View → Tool Windows:
- **bc Agents** - Shows agent status table
- **bc Channels** - Channel messages with send capability

### Commands

Access from Tools → bc menu:
- **Show Status** - Display workspace status
- **List Agents** - Show all agents
- **Agent Health** - Check agent health
- **List Channels** - Show available channels
- **Send to Channel** - Send message to channel
- **View Logs** - Show recent bc logs
- **Refresh** - Refresh status

### Keyboard Shortcuts

- `Ctrl+Alt+B` - Quick status popup

### Settings

Configure at Settings → Tools → bc:
- **bc binary path** - Path to bc binary (default: `bc`)

## Development

### Building

```bash
./gradlew buildPlugin
```

Output: `build/distributions/bc-jetbrains-*.zip`

### Testing

```bash
./gradlew runIde
```

Opens a sandboxed IDE instance with the plugin installed.

### Publishing

```bash
export PUBLISH_TOKEN=<your-token>
./gradlew publishPlugin
```

## Compatibility

- IntelliJ IDEA Community/Ultimate 2023.3+
- WebStorm 2023.3+
- GoLand 2023.3+
- PyCharm 2023.3+
- All other JetBrains IDEs 2023.3+

## License

MIT
