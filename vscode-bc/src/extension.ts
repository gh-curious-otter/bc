import * as vscode from 'vscode';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Agent data structure
interface Agent {
    name: string;
    role: string;
    state: string;
    uptime: string;
    task: string;
}

// Channel data structure
interface Channel {
    name: string;
    members: number;
}

// Process data structure
interface Process {
    pid: string;
    name: string;
    status: string;
}

// Configuration helper
function getConfig(): { binaryPath: string; refreshInterval: number; showStatusBar: boolean } {
    const config = vscode.workspace.getConfiguration('bc');
    return {
        binaryPath: config.get('binaryPath', 'bc'),
        refreshInterval: config.get('refreshInterval', 5000),
        showStatusBar: config.get('showStatusBar', true)
    };
}

// Execute bc command
async function runBcCommand(command: string): Promise<string> {
    const { binaryPath } = getConfig();
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

    try {
        const { stdout } = await execAsync(`${binaryPath} ${command}`, {
            cwd: workspaceFolder
        });
        return stdout;
    } catch (error) {
        const err = error as { stderr?: string; message?: string };
        throw new Error(err.stderr || err.message || 'Command failed');
    }
}

// Parse agent list output
function parseAgentList(output: string): Agent[] {
    const agents: Agent[] = [];
    const lines = output.split('\n');

    for (const line of lines) {
        // Match agent lines: "eng-01          engineer     working    13s                  ..."
        const match = line.match(/^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)$/);
        if (match && !line.startsWith('AGENT') && !line.startsWith('-')) {
            const [, name, role, state, uptime, task] = match;
            if (name && role && state) {
                agents.push({
                    name: name.trim(),
                    role: role.trim(),
                    state: state.trim(),
                    uptime: uptime.trim(),
                    task: task?.trim() || ''
                });
            }
        }
    }
    return agents;
}

// Parse channel list output
function parseChannelList(output: string): Channel[] {
    const channels: Channel[] = [];
    const lines = output.split('\n');

    for (const line of lines) {
        // Match channel lines
        const match = line.match(/^#?(\S+)\s+(\d+)\s+members?/i);
        if (match) {
            channels.push({
                name: match[1],
                members: parseInt(match[2], 10)
            });
        }
    }
    return channels;
}

// Parse process list output
function parseProcessList(output: string): Process[] {
    const processes: Process[] = [];
    const lines = output.split('\n');

    for (const line of lines) {
        const match = line.match(/^(\d+)\s+(\S+)\s+(\S+)/);
        if (match) {
            processes.push({
                pid: match[1],
                name: match[2],
                status: match[3]
            });
        }
    }
    return processes;
}

// Tree data provider for agents
class AgentsProvider implements vscode.TreeDataProvider<AgentItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<AgentItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: AgentItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<AgentItem[]> {
        try {
            const output = await runBcCommand('agent list --json 2>/dev/null || bc status');
            const agents = parseAgentList(output);
            return agents.map(agent => new AgentItem(agent));
        } catch {
            return [];
        }
    }
}

class AgentItem extends vscode.TreeItem {
    constructor(public readonly agent: Agent) {
        super(agent.name, vscode.TreeItemCollapsibleState.None);

        this.description = `${agent.role} - ${agent.state}`;
        this.tooltip = `${agent.name}\nRole: ${agent.role}\nState: ${agent.state}\nUptime: ${agent.uptime}\nTask: ${agent.task}`;

        // Set icon based on state
        if (agent.state === 'working' || agent.state === 'running') {
            this.iconPath = new vscode.ThemeIcon('play-circle', new vscode.ThemeColor('charts.green'));
            this.contextValue = 'agent-running';
        } else if (agent.state === 'stopped') {
            this.iconPath = new vscode.ThemeIcon('stop-circle', new vscode.ThemeColor('charts.red'));
            this.contextValue = 'agent-stopped';
        } else {
            this.iconPath = new vscode.ThemeIcon('circle-outline');
            this.contextValue = 'agent';
        }
    }
}

// Tree data provider for channels
class ChannelsProvider implements vscode.TreeDataProvider<ChannelItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<ChannelItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: ChannelItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<ChannelItem[]> {
        try {
            const output = await runBcCommand('channel list');
            const channels = parseChannelList(output);
            return channels.map(channel => new ChannelItem(channel));
        } catch {
            return [];
        }
    }
}

class ChannelItem extends vscode.TreeItem {
    constructor(public readonly channel: Channel) {
        super(`#${channel.name}`, vscode.TreeItemCollapsibleState.None);

        this.description = `${channel.members} members`;
        this.tooltip = `Channel: #${channel.name}\nMembers: ${channel.members}`;
        this.iconPath = new vscode.ThemeIcon('comment-discussion');
        this.contextValue = 'channel';

        this.command = {
            command: 'bc.channelHistory',
            title: 'View History',
            arguments: [channel.name]
        };
    }
}

// Tree data provider for processes
class ProcessesProvider implements vscode.TreeDataProvider<ProcessItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<ProcessItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: ProcessItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<ProcessItem[]> {
        try {
            const output = await runBcCommand('process list');
            const processes = parseProcessList(output);
            return processes.map(proc => new ProcessItem(proc));
        } catch {
            return [];
        }
    }
}

class ProcessItem extends vscode.TreeItem {
    constructor(public readonly process: Process) {
        super(process.name, vscode.TreeItemCollapsibleState.None);

        this.description = `PID: ${process.pid} - ${process.status}`;
        this.tooltip = `Process: ${process.name}\nPID: ${process.pid}\nStatus: ${process.status}`;
        this.iconPath = new vscode.ThemeIcon('terminal');
        this.contextValue = 'process';
    }
}

// Status bar item
let statusBarItem: vscode.StatusBarItem;
let refreshTimer: NodeJS.Timeout | undefined;

async function updateStatusBar(): Promise<void> {
    try {
        const output = await runBcCommand('status');
        // Parse summary line: "Workspace: bc-v2 | Agents: 12 | Active: 6 | Working: 6"
        const match = output.match(/Agents:\s*(\d+)\s*\|\s*Active:\s*(\d+)/);
        if (match) {
            const [, total, active] = match;
            statusBarItem.text = `$(robot) bc: ${active}/${total} agents`;
            statusBarItem.tooltip = 'Click to show bc status';
            statusBarItem.show();
        }
    } catch {
        statusBarItem.text = '$(robot) bc: not connected';
        statusBarItem.tooltip = 'bc workspace not found';
    }
}

// Extension activation
export function activate(context: vscode.ExtensionContext): void {
    console.log('bc extension is now active');

    // Create providers
    const agentsProvider = new AgentsProvider();
    const channelsProvider = new ChannelsProvider();
    const processesProvider = new ProcessesProvider();

    // Register tree views
    context.subscriptions.push(
        vscode.window.registerTreeDataProvider('bc.agentsView', agentsProvider),
        vscode.window.registerTreeDataProvider('bc.channelsView', channelsProvider),
        vscode.window.registerTreeDataProvider('bc.processesView', processesProvider)
    );

    // Create status bar item
    statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
    statusBarItem.command = 'bc.status';
    context.subscriptions.push(statusBarItem);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('bc.status', async () => {
            try {
                const output = await runBcCommand('status');
                const panel = vscode.window.createWebviewPanel(
                    'bcStatus',
                    'BC Status',
                    vscode.ViewColumn.One,
                    {}
                );
                panel.webview.html = `<pre style="font-family: monospace; white-space: pre-wrap;">${escapeHtml(output)}</pre>`;
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to get status: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.agentList', async () => {
            agentsProvider.refresh();
            vscode.window.showInformationMessage('Agent list refreshed');
        }),

        vscode.commands.registerCommand('bc.agentCreate', async () => {
            const name = await vscode.window.showInputBox({
                prompt: 'Enter agent name',
                placeHolder: 'e.g., eng-06'
            });
            if (!name) { return; }

            const role = await vscode.window.showQuickPick(
                ['engineer', 'manager', 'tech-lead', 'ux', 'product-manager'],
                { placeHolder: 'Select agent role' }
            );
            if (!role) { return; }

            try {
                await runBcCommand(`agent create ${name} --role ${role}`);
                vscode.window.showInformationMessage(`Agent ${name} created`);
                agentsProvider.refresh();
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to create agent: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.agentStop', async (item?: AgentItem) => {
            const name = item?.agent.name || await vscode.window.showInputBox({
                prompt: 'Enter agent name to stop'
            });
            if (!name) { return; }

            try {
                await runBcCommand(`agent stop ${name}`);
                vscode.window.showInformationMessage(`Agent ${name} stopped`);
                agentsProvider.refresh();
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to stop agent: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.agentStart', async (item?: AgentItem) => {
            const name = item?.agent.name || await vscode.window.showInputBox({
                prompt: 'Enter agent name to start'
            });
            if (!name) { return; }

            try {
                await runBcCommand(`agent start ${name}`);
                vscode.window.showInformationMessage(`Agent ${name} started`);
                agentsProvider.refresh();
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to start agent: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.agentAttach', async (item?: AgentItem) => {
            const name = item?.agent.name || await vscode.window.showInputBox({
                prompt: 'Enter agent name to attach'
            });
            if (!name) { return; }

            // Open terminal and attach
            const terminal = vscode.window.createTerminal(`bc: ${name}`);
            terminal.sendText(`bc attach ${name}`);
            terminal.show();
        }),

        vscode.commands.registerCommand('bc.agentPeek', async (item?: AgentItem) => {
            const name = item?.agent.name || await vscode.window.showInputBox({
                prompt: 'Enter agent name to peek'
            });
            if (!name) { return; }

            try {
                const output = await runBcCommand(`agent peek ${name}`);
                const panel = vscode.window.createWebviewPanel(
                    'bcPeek',
                    `BC: ${name} output`,
                    vscode.ViewColumn.One,
                    {}
                );
                panel.webview.html = `<pre style="font-family: monospace; white-space: pre-wrap; background: #1e1e1e; color: #d4d4d4; padding: 10px;">${escapeHtml(output)}</pre>`;
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to peek agent: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.channelSend', async () => {
            const channel = await vscode.window.showInputBox({
                prompt: 'Enter channel name',
                placeHolder: 'e.g., eng'
            });
            if (!channel) { return; }

            const message = await vscode.window.showInputBox({
                prompt: 'Enter message',
                placeHolder: 'Your message...'
            });
            if (!message) { return; }

            try {
                await runBcCommand(`channel send ${channel} "${message.replace(/"/g, '\\"')}"`);
                vscode.window.showInformationMessage(`Message sent to #${channel}`);
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to send message: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.channelHistory', async (channelName?: string) => {
            const channel = channelName || await vscode.window.showInputBox({
                prompt: 'Enter channel name',
                placeHolder: 'e.g., eng'
            });
            if (!channel) { return; }

            try {
                const output = await runBcCommand(`channel history ${channel} --limit 50`);
                const panel = vscode.window.createWebviewPanel(
                    'bcChannel',
                    `BC: #${channel}`,
                    vscode.ViewColumn.One,
                    {}
                );
                panel.webview.html = `<pre style="font-family: monospace; white-space: pre-wrap; background: #1e1e1e; color: #d4d4d4; padding: 10px;">${escapeHtml(output)}</pre>`;
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to get channel history: ${error}`);
            }
        }),

        vscode.commands.registerCommand('bc.refresh', () => {
            agentsProvider.refresh();
            channelsProvider.refresh();
            processesProvider.refresh();
            updateStatusBar();
            vscode.window.showInformationMessage('BC views refreshed');
        })
    );

    // Initial update and timer
    const config = getConfig();
    if (config.showStatusBar) {
        updateStatusBar();
        refreshTimer = setInterval(() => {
            updateStatusBar();
            agentsProvider.refresh();
        }, config.refreshInterval);
    }

    // Watch for config changes
    context.subscriptions.push(
        vscode.workspace.onDidChangeConfiguration(e => {
            if (e.affectsConfiguration('bc')) {
                const newConfig = getConfig();
                if (refreshTimer) {
                    clearInterval(refreshTimer);
                }
                if (newConfig.showStatusBar) {
                    updateStatusBar();
                    refreshTimer = setInterval(() => {
                        updateStatusBar();
                        agentsProvider.refresh();
                    }, newConfig.refreshInterval);
                } else {
                    statusBarItem.hide();
                }
            }
        })
    );
}

// Extension deactivation
export function deactivate(): void {
    if (refreshTimer) {
        clearInterval(refreshTimer);
    }
}

// HTML escape helper
function escapeHtml(text: string): string {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}
