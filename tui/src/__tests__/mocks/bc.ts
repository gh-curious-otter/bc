/**
 * Mock BC Service - Simulates bc CLI for testing
 *
 * Prevents actual CLI invocation during tests
 * Allows configuration of responses and error scenarios
 */

import type {
  Agent,
  Channel,
  ChannelMessage,
  StatusResponse,
  ChannelsResponse,
  ChannelHistory,
} from '../../types';
import {
  createMockAgents,
  createMockChannels,
  createMockMessages,
} from '../fixtures';

export interface MockBcServiceOptions {
  agents?: Agent[];
  channels?: Channel[];
  messages?: ChannelMessage[];
  shouldFail?: boolean;
  failureMessage?: string;
}

/**
 * Mock BC Service
 *
 * Simulates the bc CLI service used by hooks
 */
export class MockBcService {
  private agents: Agent[];
  private channels: Channel[];
  private messages: Map<string, ChannelMessage[]>;
  private callHistory: { command: string; args: string[] }[] = [];
  private shouldFail: boolean;
  private failureMessage: string

  constructor(options: MockBcServiceOptions = {}) {
    this.agents = options.agents ?? createMockAgents(3);
    this.channels = options.channels ?? createMockChannels(3);
    this.messages = new Map();

    // Initialize message histories for each channel
    this.channels.forEach(channel => {
      this.messages.set(channel.name, options.messages ?? createMockMessages(5));
    });

    this.shouldFail = options.shouldFail ?? false;
    this.failureMessage = options.failureMessage ?? 'Service error';
  }

  /**
   * Execute a command
   *
   * @param command Command name (e.g., 'status', 'channel')
   * @param args Command arguments
   * @returns Command result
   */
  execute(command: string, args: string[] = []): unknown {
    this.callHistory.push({ command, args });

    if (this.shouldFail) {
      throw new Error(this.failureMessage);
    }

    switch (command) {
      case 'status':
        return this.getStatus();

      case 'channel':
        if (args[0] === 'list') {
          return this.listChannels();
        }
        if (args[0] === 'history') {
          return this.getChannelHistory(args[1] ?? '');
        }
        break;

      case 'agent':
        if (args[0] === 'list') {
          return this.listAgents();
        }
        break;

      default:
        throw new Error(`Unknown command: ${command}`);
    }
  }

  /**
   * Get workspace status
   */
  private getStatus(): StatusResponse {
    const working = this.agents.filter(a => a.state === 'working').length;
    const active = this.agents.filter(a => a.state !== 'stopped').length;

    return {
      workspace: 'test-workspace',
      total: this.agents.length,
      active,
      working,
      agents: this.agents,
    };
  }

  /**
   * List all channels
   */
  private listChannels(): ChannelsResponse {
    return {
      channels: this.channels,
    };
  }

  /**
   * Get channel history
   */
  private getChannelHistory(channelName: string): ChannelHistory {
    const messages = this.messages.get(channelName) ?? [];

    return {
      channel: channelName,
      messages,
    };
  }

  /**
   * List all agents
   */
  private listAgents(): { agents: Agent[] } {
    return {
      agents: this.agents,
    };
  }

  // ========================================================================
  // Test Helpers
  // ========================================================================

  /**
   * Get call history
   */
  getCallHistory() {
    return [...this.callHistory];
  }

  /**
   * Clear call history
   */
  clearHistory() {
    this.callHistory = [];
  }

  /**
   * Assert command was called
   */
  assertCalled(command: string, args?: string[]) {
    const found = this.callHistory.find(
      call => call.command === command && (!args || this.argsMatch(call.args, args))
    );

    if (!found) {
      const callStr = args ? `${command} ${args.join(' ')}` : command;
      throw new Error(`Expected command "${callStr}" to be called, but it wasn't`);
    }
  }

  /**
   * Assert command was NOT called
   */
  assertNotCalled(command: string, args?: string[]) {
    const found = this.callHistory.find(
      call => call.command === command && (!args || this.argsMatch(call.args, args))
    );

    if (found) {
      const callStr = args ? `${command} ${args.join(' ')}` : command;
      throw new Error(`Expected command "${callStr}" not to be called, but it was`);
    }
  }

  /**
   * Get call count
   */
  getCallCount(command: string): number {
    return this.callHistory.filter(call => call.command === command).length;
  }

  /**
   * Set should fail
   */
  setShouldFail(shouldFail: boolean, message = 'Service error') {
    this.shouldFail = shouldFail;
    this.failureMessage = message;
  }

  /**
   * Add agent
   */
  addAgent(agent: Agent) {
    this.agents.push(agent);
  }

  /**
   * Remove agent by name
   */
  removeAgent(name: string) {
    this.agents = this.agents.filter(a => a.name !== name);
  }

  /**
   * Get agent
   */
  getAgent(name: string): Agent | undefined {
    return this.agents.find(a => a.name === name);
  }

  /**
   * Update agent
   */
  updateAgent(name: string, updates: Partial<Agent>) {
    const agent = this.getAgent(name);
    if (agent) {
      Object.assign(agent, updates);
    }
  }

  /**
   * Add channel
   */
  addChannel(channel: Channel) {
    this.channels.push(channel);
    this.messages.set(channel.name, []);
  }

  /**
   * Add message to channel
   */
  addMessage(channelName: string, message: ChannelMessage) {
    const messages = this.messages.get(channelName) ?? [];
    messages.push(message);
    this.messages.set(channelName, messages);
  }

  // ========================================================================
  // Helpers
  // ========================================================================

  private argsMatch(actual: string[], expected: string[]): boolean {
    if (actual.length !== expected.length) return false;
    return actual.every((arg, i) => arg === expected[i]);
  }
}

/**
 * Create mock bc service with defaults
 */
export function createMockBcService(
  options: MockBcServiceOptions = {}
): MockBcService {
  return new MockBcService(options);
}

export default {
  MockBcService,
  createMockBcService,
};
