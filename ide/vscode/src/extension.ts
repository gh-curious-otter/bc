import * as vscode from 'vscode';
import { BcService } from './services/bcService';
import { AgentsTreeProvider } from './views/agentsTreeProvider';
import { ChannelsTreeProvider } from './views/channelsTreeProvider';
import { StatusBar } from './views/statusBar';

let bcService: BcService;
let statusBar: StatusBar;
let refreshInterval: NodeJS.Timeout | undefined;

export function activate(context: vscode.ExtensionContext) {
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    if (!workspaceFolder) {
        return;
    }

    bcService = new BcService(workspaceFolder.uri.fsPath);

    if (!bcService.isWorkspace()) {
        return;
    }

    console.log('bc extension activated');

    // Create tree providers
    const agentsProvider = new AgentsTreeProvider(bcService);
    const channelsProvider = new ChannelsTreeProvider(bcService);

    // Register tree views
    vscode.window.registerTreeDataProvider('bc-agents', agentsProvider);
    vscode.window.registerTreeDataProvider('bc-channels', channelsProvider);

    // Create status bar
    statusBar = new StatusBar(bcService);
    context.subscriptions.push(statusBar);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('bc.status', () => showStatus()),
        vscode.commands.registerCommand('bc.agentList', () => showAgentList()),
        vscode.commands.registerCommand('bc.agentHealth', () => showAgentHealth()),
        vscode.commands.registerCommand('bc.channelList', () => showChannelList()),
        vscode.commands.registerCommand('bc.channelSend', () => sendToChannel()),
        vscode.commands.registerCommand('bc.logs', () => showLogs()),
        vscode.commands.registerCommand('bc.refresh', () => {
            agentsProvider.refresh();
            channelsProvider.refresh();
            statusBar.refresh();
        })
    );

    // Set up auto-refresh
    const config = vscode.workspace.getConfiguration('bc');
    const interval = config.get<number>('refreshInterval', 30);
    if (interval > 0) {
        refreshInterval = setInterval(() => {
            agentsProvider.refresh();
            channelsProvider.refresh();
            statusBar.refresh();
        }, interval * 1000);
    }

    // Initial refresh
    statusBar.refresh();
}

export function deactivate() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
}

async function showStatus() {
    const status = await bcService.getStatus();
    if (status) {
        vscode.window.showInformationMessage(
            `bc: ${status.workspace} | Agents: ${status.agentCount} | Active: ${status.activeCount} | Working: ${status.workingCount}`
        );
    } else {
        vscode.window.showWarningMessage('Failed to get bc status');
    }
}

async function showAgentList() {
    const agents = await bcService.listAgents();
    if (agents.length === 0) {
        vscode.window.showInformationMessage('No agents found');
        return;
    }

    const items = agents.map(a => ({
        label: a.name,
        description: `${a.role} - ${a.state}`,
        detail: a.task || undefined
    }));

    const selected = await vscode.window.showQuickPick(items, {
        placeHolder: 'Select an agent'
    });

    if (selected) {
        vscode.window.showInformationMessage(`Agent: ${selected.label}`);
    }
}

async function showAgentHealth() {
    const output = await bcService.execute('agent', 'health');
    if (output) {
        const doc = await vscode.workspace.openTextDocument({
            content: output,
            language: 'plaintext'
        });
        vscode.window.showTextDocument(doc);
    }
}

async function showChannelList() {
    const channels = await bcService.listChannels();
    if (channels.length === 0) {
        vscode.window.showInformationMessage('No channels found');
        return;
    }

    const selected = await vscode.window.showQuickPick(channels, {
        placeHolder: 'Select a channel'
    });

    if (selected) {
        const history = await bcService.getChannelHistory(selected);
        const content = history.map(m => `[${m.timestamp}] ${m.sender}: ${m.message}`).join('\n');
        const doc = await vscode.workspace.openTextDocument({
            content: content || 'No messages',
            language: 'plaintext'
        });
        vscode.window.showTextDocument(doc);
    }
}

async function sendToChannel() {
    const channels = await bcService.listChannels();
    if (channels.length === 0) {
        vscode.window.showWarningMessage('No channels available');
        return;
    }

    const channel = await vscode.window.showQuickPick(channels, {
        placeHolder: 'Select channel'
    });

    if (!channel) {
        return;
    }

    const message = await vscode.window.showInputBox({
        prompt: `Message for #${channel}`,
        placeHolder: 'Enter message'
    });

    if (message) {
        const success = await bcService.sendToChannel(channel, message);
        if (success) {
            vscode.window.showInformationMessage(`Sent to #${channel}`);
        } else {
            vscode.window.showErrorMessage('Failed to send message');
        }
    }
}

async function showLogs() {
    const logs = await bcService.getLogs();
    if (logs) {
        const doc = await vscode.workspace.openTextDocument({
            content: logs,
            language: 'plaintext'
        });
        vscode.window.showTextDocument(doc);
    }
}
