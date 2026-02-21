import { execSync, exec } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import * as vscode from 'vscode';

export interface BcStatus {
    workspace: string;
    agentCount: number;
    activeCount: number;
    workingCount: number;
}

export interface Agent {
    name: string;
    role: string;
    state: string;
    uptime: string;
    task: string;
}

export interface ChannelMessage {
    timestamp: string;
    sender: string;
    message: string;
}

export class BcService {
    private workspacePath: string;
    private bcPath: string;
    private _isWorkspace: boolean;

    constructor(workspacePath: string) {
        this.workspacePath = workspacePath;
        this.bcPath = vscode.workspace.getConfiguration('bc').get('binaryPath', 'bc');
        this._isWorkspace = this.detectWorkspace();
    }

    private detectWorkspace(): boolean {
        const bcDir = path.join(this.workspacePath, '.bc');
        return fs.existsSync(bcDir) && fs.statSync(bcDir).isDirectory();
    }

    isWorkspace(): boolean {
        return this._isWorkspace;
    }

    async execute(...args: string[]): Promise<string | null> {
        if (!this._isWorkspace) {
            return null;
        }

        return new Promise((resolve) => {
            const cmd = `${this.bcPath} ${args.join(' ')}`;
            exec(cmd, {
                cwd: this.workspacePath,
                env: { ...process.env, NO_COLOR: '1' },
                timeout: 30000
            }, (error, stdout) => {
                if (error) {
                    console.warn(`bc command failed: ${args.join(' ')}`, error);
                    resolve(null);
                } else {
                    resolve(stdout);
                }
            });
        });
    }

    async getStatus(): Promise<BcStatus | null> {
        const output = await this.execute('status', '--json');
        if (!output) {
            return null;
        }

        try {
            const data = JSON.parse(output);
            return {
                workspace: data.workspace || 'unknown',
                agentCount: data.agent_count || 0,
                activeCount: data.active_count || 0,
                workingCount: data.working_count || 0
            };
        } catch {
            return null;
        }
    }

    async listAgents(): Promise<Agent[]> {
        const output = await this.execute('agent', 'list', '--json');
        if (!output) {
            return [];
        }

        try {
            const data = JSON.parse(output);
            if (Array.isArray(data)) {
                return data.map((a: Record<string, unknown>) => ({
                    name: String(a.name || ''),
                    role: String(a.role || ''),
                    state: String(a.state || ''),
                    uptime: String(a.uptime || '-'),
                    task: String(a.task || '')
                }));
            }
        } catch {
            // Parse table output as fallback
            return this.parseAgentTable(output);
        }
        return [];
    }

    private parseAgentTable(output: string): Agent[] {
        return output.split('\n')
            .filter(line => line.includes('engineer') || line.includes('manager') || line.includes('root'))
            .map(line => {
                const parts = line.split(/\s{2,}/).map(p => p.trim());
                if (parts.length >= 4) {
                    return {
                        name: parts[0],
                        role: parts[1],
                        state: parts[2],
                        uptime: parts[3] || '-',
                        task: parts[4] || ''
                    };
                }
                return null;
            })
            .filter((a): a is Agent => a !== null);
    }

    async listChannels(): Promise<string[]> {
        const output = await this.execute('channel', 'list');
        if (!output) {
            return [];
        }

        return output.split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0 && !line.startsWith('CHANNEL'));
    }

    async getChannelHistory(channel: string, limit: number = 20): Promise<ChannelMessage[]> {
        const output = await this.execute('channel', 'history', channel, '--limit', String(limit));
        if (!output) {
            return [];
        }

        const messages: ChannelMessage[] = [];
        const regex = /\[(\d+)\]\s*\[([^\]]+)\]\s*([^:]+):\s*(.*)/;

        for (const line of output.split('\n')) {
            const match = regex.exec(line);
            if (match) {
                messages.push({
                    timestamp: match[2],
                    sender: match[3].trim(),
                    message: match[4].trim()
                });
            }
        }

        return messages;
    }

    async sendToChannel(channel: string, message: string): Promise<boolean> {
        const output = await this.execute('channel', 'send', channel, message);
        return output !== null;
    }

    async getLogs(limit: number = 50): Promise<string> {
        const output = await this.execute('logs', '--tail', String(limit));
        return output || '';
    }
}
