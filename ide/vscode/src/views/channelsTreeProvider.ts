import * as vscode from 'vscode';
import { BcService } from '../services/bcService';

export class ChannelsTreeProvider implements vscode.TreeDataProvider<ChannelItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<ChannelItem | undefined | null | void> = new vscode.EventEmitter<ChannelItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<ChannelItem | undefined | null | void> = this._onDidChangeTreeData.event;

    private channels: string[] = [];

    constructor(private bcService: BcService) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: ChannelItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<ChannelItem[]> {
        this.channels = await this.bcService.listChannels();
        return this.channels.map(channel => new ChannelItem(channel, this.bcService));
    }
}

class ChannelItem extends vscode.TreeItem {
    constructor(channel: string, private bcService: BcService) {
        super(`#${channel}`, vscode.TreeItemCollapsibleState.None);

        this.tooltip = `Channel: ${channel}`;
        this.iconPath = new vscode.ThemeIcon('comment-discussion');
        this.contextValue = 'channel';

        this.command = {
            command: 'bc.channelSend',
            title: 'Send Message',
            arguments: [channel]
        };
    }
}
