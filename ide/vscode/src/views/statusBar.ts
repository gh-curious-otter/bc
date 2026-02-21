import * as vscode from 'vscode';
import { BcService } from '../services/bcService';

export class StatusBar implements vscode.Disposable {
    private statusBarItem: vscode.StatusBarItem;
    private bcService: BcService;

    constructor(bcService: BcService) {
        this.bcService = bcService;
        this.statusBarItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Left,
            100
        );
        this.statusBarItem.command = 'bc.status';
        this.statusBarItem.tooltip = 'bc workspace status';
        this.statusBarItem.show();
    }

    async refresh(): Promise<void> {
        const status = await this.bcService.getStatus();
        if (status) {
            const working = status.workingCount > 0 ? `$(sync~spin) ${status.workingCount}` : '';
            this.statusBarItem.text = `$(robot) bc: ${status.activeCount}/${status.agentCount} ${working}`;
        } else {
            this.statusBarItem.text = '$(robot) bc';
        }
    }

    dispose(): void {
        this.statusBarItem.dispose();
    }
}
