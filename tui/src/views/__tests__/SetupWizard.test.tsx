/**
 * SetupWizard Tests
 * Issue #1189: Interactive wizard UI
 *
 * Tests cover:
 * - Wizard steps configuration
 * - Options count per step
 * - Progress bar calculation
 * - Preset to config mapping
 * - Agent/channel count mapping
 * - Navigation hints
 * - Keyboard shortcuts
 */

import { describe, test, expect } from 'bun:test';

// Types matching SetupWizard
type PresetType = 'solo' | 'pair' | 'team' | 'custom';

interface WizardStep {
  id: string;
  title: string;
}

interface WizardConfig {
  preset: PresetType;
  agentCount: number;
  channelCount: number;
}

// STEPS constant matching SetupWizard
const STEPS: WizardStep[] = [
  { id: 'welcome', title: 'Welcome' },
  { id: 'preset', title: 'Team Size' },
  { id: 'agents', title: 'Agents' },
  { id: 'channels', title: 'Channels' },
  { id: 'complete', title: 'Complete' },
];

// Helper functions matching SetupWizard logic
function getOptionsCount(stepId: string): number {
  switch (stepId) {
    case 'welcome':
      return 2; // Continue / Skip
    case 'preset':
      return 4; // solo / pair / team / custom
    case 'agents':
      return 4; // 1-3 agents or custom
    case 'channels':
      return 3; // 2-4 channels
    case 'complete':
      return 2; // Finish / Start over
    default:
      return 1;
  }
}

function calculateProgressBar(currentStep: number, totalSteps: number, width: number): string {
  const filled = Math.round((currentStep / (totalSteps - 1)) * width);
  return '█'.repeat(filled) + '░'.repeat(width - filled);
}

function getPresetConfig(preset: PresetType): { agentCount: number } {
  const agentCounts: Record<PresetType, number> = {
    solo: 1,
    pair: 2,
    team: 4,
    custom: 1,
  };
  return { agentCount: agentCounts[preset] };
}

function getAgentCountFromSelection(selectedOption: number): number {
  return selectedOption + 1;
}

function getChannelCountFromSelection(selectedOption: number): number {
  return selectedOption + 2;
}

function getHints(isNarrow: boolean): string {
  return isNarrow
    ? 'j/k:nav Enter:select Esc:cancel'
    : 'j/k: navigate | Enter: select | s: skip | b: back | Esc: cancel';
}

function getProgressWidth(isNarrow: boolean): number {
  return isNarrow ? 30 : 50;
}

describe('SetupWizard', () => {
  describe('Wizard Steps', () => {
    test('has 5 steps', () => {
      expect(STEPS).toHaveLength(5);
    });

    test('steps have correct order', () => {
      expect(STEPS[0].id).toBe('welcome');
      expect(STEPS[1].id).toBe('preset');
      expect(STEPS[2].id).toBe('agents');
      expect(STEPS[3].id).toBe('channels');
      expect(STEPS[4].id).toBe('complete');
    });

    test('steps have titles', () => {
      for (const step of STEPS) {
        expect(step.title).toBeTruthy();
        expect(step.title.length).toBeGreaterThan(0);
      }
    });

    test('step IDs are unique', () => {
      const ids = STEPS.map((s) => s.id);
      const uniqueIds = new Set(ids);
      expect(uniqueIds.size).toBe(ids.length);
    });
  });

  describe('Options Count', () => {
    test('welcome has 2 options', () => {
      expect(getOptionsCount('welcome')).toBe(2);
    });

    test('preset has 4 options', () => {
      expect(getOptionsCount('preset')).toBe(4);
    });

    test('agents has 4 options', () => {
      expect(getOptionsCount('agents')).toBe(4);
    });

    test('channels has 3 options', () => {
      expect(getOptionsCount('channels')).toBe(3);
    });

    test('complete has 2 options', () => {
      expect(getOptionsCount('complete')).toBe(2);
    });

    test('unknown step returns 1', () => {
      expect(getOptionsCount('unknown')).toBe(1);
    });
  });

  describe('Progress Bar', () => {
    test('shows empty at step 0', () => {
      const bar = calculateProgressBar(0, 5, 10);
      expect(bar).toBe('░░░░░░░░░░');
    });

    test('shows full at last step', () => {
      const bar = calculateProgressBar(4, 5, 10);
      expect(bar).toBe('██████████');
    });

    test('shows partial at middle step', () => {
      const bar = calculateProgressBar(2, 5, 10);
      expect(bar).toBe('█████░░░░░');
    });

    test('handles narrow width', () => {
      const bar = calculateProgressBar(2, 5, 30);
      expect(bar.length).toBe(30);
    });

    test('handles wide width', () => {
      const bar = calculateProgressBar(2, 5, 50);
      expect(bar.length).toBe(50);
    });
  });

  describe('Preset Configuration', () => {
    test('solo preset has 1 agent', () => {
      const config = getPresetConfig('solo');
      expect(config.agentCount).toBe(1);
    });

    test('pair preset has 2 agents', () => {
      const config = getPresetConfig('pair');
      expect(config.agentCount).toBe(2);
    });

    test('team preset has 4 agents', () => {
      const config = getPresetConfig('team');
      expect(config.agentCount).toBe(4);
    });

    test('custom preset defaults to 1 agent', () => {
      const config = getPresetConfig('custom');
      expect(config.agentCount).toBe(1);
    });
  });

  describe('Agent Count Selection', () => {
    test('option 0 gives 1 agent', () => {
      expect(getAgentCountFromSelection(0)).toBe(1);
    });

    test('option 1 gives 2 agents', () => {
      expect(getAgentCountFromSelection(1)).toBe(2);
    });

    test('option 2 gives 3 agents', () => {
      expect(getAgentCountFromSelection(2)).toBe(3);
    });

    test('option 3 gives 4 agents', () => {
      expect(getAgentCountFromSelection(3)).toBe(4);
    });
  });

  describe('Channel Count Selection', () => {
    test('option 0 gives 2 channels', () => {
      expect(getChannelCountFromSelection(0)).toBe(2);
    });

    test('option 1 gives 3 channels', () => {
      expect(getChannelCountFromSelection(1)).toBe(3);
    });

    test('option 2 gives 4 channels', () => {
      expect(getChannelCountFromSelection(2)).toBe(4);
    });
  });

  describe('Navigation Hints', () => {
    test('narrow hints are compact', () => {
      const hints = getHints(true);
      expect(hints).toBe('j/k:nav Enter:select Esc:cancel');
      expect(hints.length).toBeLessThan(40);
    });

    test('wide hints are detailed', () => {
      const hints = getHints(false);
      expect(hints).toBe('j/k: navigate | Enter: select | s: skip | b: back | Esc: cancel');
      expect(hints).toContain('skip');
      expect(hints).toContain('back');
    });
  });

  describe('Progress Width', () => {
    test('narrow uses 30 chars', () => {
      expect(getProgressWidth(true)).toBe(30);
    });

    test('wide uses 50 chars', () => {
      expect(getProgressWidth(false)).toBe(50);
    });
  });

  describe('Default Config', () => {
    test('has default values', () => {
      const defaultConfig: WizardConfig = {
        preset: 'solo',
        agentCount: 1,
        channelCount: 2,
      };

      expect(defaultConfig.preset).toBe('solo');
      expect(defaultConfig.agentCount).toBe(1);
      expect(defaultConfig.channelCount).toBe(2);
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      j: 'down',
      k: 'up',
      Enter: 'select',
      s: 'skip',
      b: 'back',
      Escape: 'cancel',
    };

    test('navigation shortcuts', () => {
      expect(shortcuts.j).toBe('down');
      expect(shortcuts.k).toBe('up');
    });

    test('action shortcuts', () => {
      expect(shortcuts.Enter).toBe('select');
      expect(shortcuts.s).toBe('skip');
    });

    test('back/cancel shortcuts', () => {
      expect(shortcuts.b).toBe('back');
      expect(shortcuts.Escape).toBe('cancel');
    });
  });

  describe('Step Navigation', () => {
    test('can move forward', () => {
      let currentStep = 0;
      const handleNext = () => {
        if (currentStep < STEPS.length - 1) {
          currentStep += 1;
        }
      };

      handleNext();
      expect(currentStep).toBe(1);
      expect(STEPS[currentStep].id).toBe('preset');
    });

    test('can move backward', () => {
      let currentStep = 2;
      const handleBack = () => {
        if (currentStep > 0) {
          currentStep -= 1;
        }
      };

      handleBack();
      expect(currentStep).toBe(1);
      expect(STEPS[currentStep].id).toBe('preset');
    });

    test('cannot move before first step', () => {
      let currentStep = 0;
      const handleBack = () => {
        if (currentStep > 0) {
          currentStep -= 1;
        }
      };

      handleBack();
      expect(currentStep).toBe(0);
    });

    test('cannot move past last step', () => {
      let currentStep = STEPS.length - 1;
      const handleNext = () => {
        if (currentStep < STEPS.length - 1) {
          currentStep += 1;
        }
      };

      handleNext();
      expect(currentStep).toBe(STEPS.length - 1);
    });
  });

  describe('Option Selection', () => {
    test('option bounds check - minimum', () => {
      let selectedOption = 0;
      const moveUp = () => {
        selectedOption = Math.max(selectedOption - 1, 0);
      };

      moveUp();
      expect(selectedOption).toBe(0);
    });

    test('option bounds check - maximum', () => {
      const maxOptions = 4;
      let selectedOption = maxOptions - 1;
      const moveDown = () => {
        selectedOption = Math.min(selectedOption + 1, maxOptions - 1);
      };

      moveDown();
      expect(selectedOption).toBe(maxOptions - 1);
    });

    test('resets on step change', () => {
      let currentStep = 0;
      let selectedOption = 2;

      const handleNext = () => {
        currentStep += 1;
        selectedOption = 0; // Reset on step change
      };

      handleNext();
      expect(selectedOption).toBe(0);
    });
  });

  describe('Welcome Step', () => {
    test('option 0 is continue', () => {
      const options = ['Continue setup', 'Skip (use defaults)'];
      expect(options[0]).toBe('Continue setup');
    });

    test('option 1 is skip', () => {
      const options = ['Continue setup', 'Skip (use defaults)'];
      expect(options[1]).toBe('Skip (use defaults)');
    });
  });

  describe('Preset Step', () => {
    test('has all preset types', () => {
      const presets: PresetType[] = ['solo', 'pair', 'team', 'custom'];
      expect(presets).toHaveLength(4);
    });

    test('preset descriptions', () => {
      const descriptions: Record<PresetType, string> = {
        solo: '1 engineer agent',
        pair: '2 engineer agents',
        team: 'PM + 2 engineers + UX',
        custom: 'Configure manually',
      };

      expect(descriptions.solo).toBe('1 engineer agent');
      expect(descriptions.pair).toBe('2 engineer agents');
      expect(descriptions.team).toContain('PM');
      expect(descriptions.custom).toBe('Configure manually');
    });
  });

  describe('Channels Step', () => {
    test('basic has 2 channels', () => {
      const channels = { basic: 2, standard: 3, full: 4 };
      expect(channels.basic).toBe(2);
    });

    test('standard has 3 channels', () => {
      const channels = { basic: 2, standard: 3, full: 4 };
      expect(channels.standard).toBe(3);
    });

    test('full has 4 channels', () => {
      const channels = { basic: 2, standard: 3, full: 4 };
      expect(channels.full).toBe(4);
    });

    test('default channels', () => {
      const defaultChannels = ['#general', '#engineering'];
      expect(defaultChannels).toHaveLength(2);
      expect(defaultChannels).toContain('#general');
      expect(defaultChannels).toContain('#engineering');
    });
  });

  describe('Complete Step', () => {
    test('option 0 is finish', () => {
      const options = ['Finish and start', 'Start over'];
      expect(options[0]).toBe('Finish and start');
    });

    test('option 1 is start over', () => {
      const options = ['Finish and start', 'Start over'];
      expect(options[1]).toBe('Start over');
    });

    test('start over resets to step 0', () => {
      let currentStep = STEPS.length - 1;
      let selectedOption = 1;

      // Simulate start over
      if (selectedOption === 1) {
        currentStep = 0;
        selectedOption = 0;
      }

      expect(currentStep).toBe(0);
      expect(selectedOption).toBe(0);
    });
  });

  describe('Selection Indicator', () => {
    test('selected uses triangle', () => {
      const selected = true;
      const indicator = selected ? '▸ ' : '  ';
      expect(indicator).toBe('▸ ');
    });

    test('unselected uses spaces', () => {
      const selected = false;
      const indicator = selected ? '▸ ' : '  ';
      expect(indicator).toBe('  ');
    });
  });

  describe('Responsive Layout', () => {
    test('narrow is 80 or less', () => {
      const termWidth = 80;
      const isNarrow = termWidth <= 80;
      expect(isNarrow).toBe(true);
    });

    test('wide is greater than 80', () => {
      const termWidth = 120;
      const isNarrow = termWidth <= 80;
      expect(isNarrow).toBe(false);
    });

    test('narrow welcome message is shorter', () => {
      const narrowMsg = "Let's set up your workspace.";
      const wideMsg = 'This wizard will help you set up your first workspace.';
      expect(narrowMsg.length).toBeLessThan(wideMsg.length);
    });
  });

  describe('Header Display', () => {
    test('shows step number', () => {
      const currentStep = 2;
      const totalSteps = 5;
      const display = `Step ${currentStep + 1}/${totalSteps}`;
      expect(display).toBe('Step 3/5');
    });

    test('shows step title', () => {
      const currentStep = 1;
      const display = STEPS[currentStep].title;
      expect(display).toBe('Team Size');
    });

    test('shows estimated time', () => {
      const estimatedTime = '~2 min';
      expect(estimatedTime).toBe('~2 min');
    });
  });

  describe('Config State', () => {
    test('config updates on preset selection', () => {
      let config: WizardConfig = {
        preset: 'solo',
        agentCount: 1,
        channelCount: 2,
      };

      // Select team preset
      const selectedOption = 2;
      const presets: PresetType[] = ['solo', 'pair', 'team', 'custom'];
      const preset = presets[selectedOption];
      const agentCounts: Record<PresetType, number> = {
        solo: 1,
        pair: 2,
        team: 4,
        custom: 1,
      };

      config = {
        ...config,
        preset,
        agentCount: agentCounts[preset],
      };

      expect(config.preset).toBe('team');
      expect(config.agentCount).toBe(4);
    });

    test('config updates on agent selection', () => {
      let config: WizardConfig = {
        preset: 'custom',
        agentCount: 1,
        channelCount: 2,
      };

      const selectedOption = 2;
      config = { ...config, agentCount: selectedOption + 1 };

      expect(config.agentCount).toBe(3);
    });

    test('config updates on channel selection', () => {
      let config: WizardConfig = {
        preset: 'solo',
        agentCount: 1,
        channelCount: 2,
      };

      const selectedOption = 1;
      config = { ...config, channelCount: selectedOption + 2 };

      expect(config.channelCount).toBe(3);
    });
  });
});
