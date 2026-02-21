import * as vscode from 'vscode';
import { BcService, Agent } from '../services/bcService';

export class AgentsTreeProvider implements vscode.TreeDataProvider<AgentItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<AgentItem | undefined | null | void> = new vscode.EventEmitter<AgentItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<AgentItem | undefined | null | void> = this._onDidChangeTreeData.event;

    private agents: Agent[] = [];

    constructor(private bcService: BcService) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: AgentItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<AgentItem[]> {
        this.agents = await this.bcService.listAgents();
        return this.agents.map(agent => new AgentItem(agent));
    }
}

class AgentItem extends vscode.TreeItem {
    constructor(agent: Agent) {
        super(agent.name, vscode.TreeItemCollapsibleState.None);

        this.description = `${agent.role} - ${agent.state}`;
        this.tooltip = agent.task || `${agent.name} (${agent.role})`;

        // Set icon based on state
        const iconColor = this.getIconColor(agent.state);
        this.iconPath = new vscode.ThemeIcon('circle-filled', iconColor);

        this.contextValue = 'agent';
    }

    private getIconColor(state: string): vscode.ThemeColor {
        switch (state.toLowerCase()) {
            case 'working':
                return new vscode.ThemeColor('testing.iconPassed');
            case 'active':
                return new vscode.ThemeColor('charts.green');
            case 'idle':
                return new vscode.ThemeColor('charts.yellow');
            case 'stopped':
                return new vscode.ThemeColor('charts.red');
            default:
                return new vscode.ThemeColor('foreground');
        }
    }
}
