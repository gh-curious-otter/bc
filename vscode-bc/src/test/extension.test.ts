import * as assert from 'assert';

// Test parsing functions - extracted for testability

interface Agent {
    name: string;
    role: string;
    state: string;
    uptime: string;
    task: string;
}

interface Channel {
    name: string;
    members: number;
}

// Parse agent list output
function parseAgentList(output: string): Agent[] {
    const agents: Agent[] = [];
    const lines = output.split('\n');

    for (const line of lines) {
        const match = line.match(/^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)$/);
        // Skip header lines: "AGENT...", "---...", "───...", "Workspace:..."
        if (match && !line.startsWith('AGENT') && !line.startsWith('-') && !line.startsWith('─') && !line.startsWith('Workspace')) {
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

// HTML escape helper
function escapeHtml(text: string): string {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

// Tests
describe('Extension Parsing Tests', () => {
    describe('parseAgentList', () => {
        it('should parse agent list output', () => {
            const output = `Workspace: bc-v2 | Agents: 12 | Active: 6 | Working: 6
────────────────────────────────────────────────────────────

AGENT           ROLE         STATE      UPTIME               TASK
--------------------------------------------------------------------------------
root            root         working    26h 33m              Dilly-dallying
eng-01          engineer     working    13s                  Implementing
eng-02          engineer     stopped    -                    Waiting`;

            const agents = parseAgentList(output);

            assert.strictEqual(agents.length, 3);
            assert.strictEqual(agents[0].name, 'root');
            assert.strictEqual(agents[0].role, 'root');
            assert.strictEqual(agents[0].state, 'working');
            assert.strictEqual(agents[1].name, 'eng-01');
            assert.strictEqual(agents[1].role, 'engineer');
            assert.strictEqual(agents[2].state, 'stopped');
        });

        it('should handle empty output', () => {
            const agents = parseAgentList('');
            assert.strictEqual(agents.length, 0);
        });

        it('should skip header lines', () => {
            const output = `AGENT           ROLE         STATE      UPTIME               TASK
--------------------------------------------------------------------------------`;
            const agents = parseAgentList(output);
            assert.strictEqual(agents.length, 0);
        });
    });

    describe('parseChannelList', () => {
        it('should parse channel list output', () => {
            const output = `#eng        7 members
#pr         5 members
#standup    4 members`;

            const channels = parseChannelList(output);

            assert.strictEqual(channels.length, 3);
            assert.strictEqual(channels[0].name, 'eng');
            assert.strictEqual(channels[0].members, 7);
            assert.strictEqual(channels[1].name, 'pr');
            assert.strictEqual(channels[1].members, 5);
        });

        it('should handle single member', () => {
            const output = `#solo    1 member`;
            const channels = parseChannelList(output);
            assert.strictEqual(channels.length, 1);
            assert.strictEqual(channels[0].members, 1);
        });

        it('should handle empty output', () => {
            const channels = parseChannelList('');
            assert.strictEqual(channels.length, 0);
        });
    });

    describe('escapeHtml', () => {
        it('should escape HTML special characters', () => {
            assert.strictEqual(escapeHtml('<script>'), '&lt;script&gt;');
            assert.strictEqual(escapeHtml('a & b'), 'a &amp; b');
            assert.strictEqual(escapeHtml('"quoted"'), '&quot;quoted&quot;');
            assert.strictEqual(escapeHtml("'single'"), '&#039;single&#039;');
        });

        it('should handle empty string', () => {
            assert.strictEqual(escapeHtml(''), '');
        });

        it('should handle text without special chars', () => {
            assert.strictEqual(escapeHtml('hello world'), 'hello world');
        });
    });
});
