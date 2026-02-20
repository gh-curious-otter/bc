/**
 * SetupWizard - First-run TUI wizard for workspace setup
 * Issue #1189: Interactive wizard UI (Ink)
 *
 * UX Requirements:
 * - ≤5 steps with progress indicator
 * - Keyboard navigation (j/k, Enter, Escape)
 * - Skip/cancel at each step
 * - 80x24 responsive design
 * - Estimated time display
 */

import React, { useState, useCallback } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';

type PresetType = 'solo' | 'pair' | 'team' | 'custom';

interface WizardStep {
  id: string;
  title: string;
}

const STEPS: WizardStep[] = [
  { id: 'welcome', title: 'Welcome' },
  { id: 'preset', title: 'Team Size' },
  { id: 'agents', title: 'Agents' },
  { id: 'channels', title: 'Channels' },
  { id: 'complete', title: 'Complete' },
];

interface SetupWizardProps {
  /** Callback when wizard completes */
  onComplete?: (config: WizardConfig) => void;
  /** Callback when wizard is cancelled */
  onCancel?: () => void;
  /** Disable input handling (for testing) */
  disableInput?: boolean;
}

interface WizardConfig {
  preset: PresetType;
  agentCount: number;
  channelCount: number;
}

export function SetupWizard({
  onComplete,
  onCancel,
  disableInput = false,
}: SetupWizardProps): React.ReactElement {
  const { stdout } = useStdout();
  const [currentStep, setCurrentStep] = useState(0);
  const [selectedOption, setSelectedOption] = useState(0);
  const [config, setConfig] = useState<WizardConfig>({
    preset: 'solo',
    agentCount: 1,
    channelCount: 2,
  });

  // Terminal dimensions for responsive layout
  const termWidth = stdout.columns || 80;
  const isNarrow = termWidth <= 80;

  // Handle keyboard navigation
  useInput(
    (input, key) => {
      if (disableInput) return;

      // Cancel with Escape
      if (key.escape) {
        onCancel?.();
        return;
      }

      // Skip with 's'
      if (input === 's' && currentStep < STEPS.length - 1) {
        handleNext();
        return;
      }

      // Navigate options with j/k or arrows
      if (input === 'j' || key.downArrow) {
        setSelectedOption((prev) => Math.min(prev + 1, getOptionsCount() - 1));
      }
      if (input === 'k' || key.upArrow) {
        setSelectedOption((prev) => Math.max(prev - 1, 0));
      }

      // Select with Enter
      if (key.return) {
        handleSelect();
      }

      // Go back with 'b' or left arrow
      if (input === 'b' || key.leftArrow) {
        if (currentStep > 0) {
          setCurrentStep((prev) => prev - 1);
          setSelectedOption(0);
        }
      }
    },
    { isActive: !disableInput }
  );

  const getOptionsCount = useCallback((): number => {
    switch (STEPS[currentStep].id) {
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
  }, [currentStep]);

  const handleNext = useCallback(() => {
    if (currentStep < STEPS.length - 1) {
      setCurrentStep((prev) => prev + 1);
      setSelectedOption(0);
    }
  }, [currentStep]);

  const handleSelect = useCallback(() => {
    const stepId = STEPS[currentStep].id;

    switch (stepId) {
      case 'welcome':
        if (selectedOption === 1) {
          // Skip setup
          onComplete?.(config);
        } else {
          handleNext();
        }
        break;

      case 'preset': {
        const presets: PresetType[] = ['solo', 'pair', 'team', 'custom'];
        const preset = presets[selectedOption];
        const agentCounts = { solo: 1, pair: 2, team: 4, custom: 1 };
        setConfig((prev) => ({
          ...prev,
          preset,
          agentCount: agentCounts[preset],
        }));
        handleNext();
        break;
      }

      case 'agents':
        setConfig((prev) => ({ ...prev, agentCount: selectedOption + 1 }));
        handleNext();
        break;

      case 'channels':
        setConfig((prev) => ({ ...prev, channelCount: selectedOption + 2 }));
        handleNext();
        break;

      case 'complete':
        if (selectedOption === 0) {
          onComplete?.(config);
        } else {
          // Start over
          setCurrentStep(0);
          setSelectedOption(0);
        }
        break;
    }
  }, [currentStep, selectedOption, config, handleNext, onComplete]);

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header with progress */}
      <WizardHeader
        currentStep={currentStep}
        totalSteps={STEPS.length}
        stepTitle={STEPS[currentStep].title}
        isNarrow={isNarrow}
      />

      {/* Step content */}
      <Box marginTop={1} marginBottom={1} flexDirection="column">
        <StepContent
          step={STEPS[currentStep].id}
          selectedOption={selectedOption}
          config={config}
          isNarrow={isNarrow}
        />
      </Box>

      {/* Footer with hints */}
      <WizardFooter isNarrow={isNarrow} />
    </Box>
  );
}

interface WizardHeaderProps {
  currentStep: number;
  totalSteps: number;
  stepTitle: string;
  isNarrow: boolean;
}

function WizardHeader({
  currentStep,
  totalSteps,
  stepTitle,
  isNarrow,
}: WizardHeaderProps): React.ReactElement {
  // Progress bar
  const progressWidth = isNarrow ? 30 : 50;
  const filled = Math.round((currentStep / (totalSteps - 1)) * progressWidth);
  const progressBar = '█'.repeat(filled) + '░'.repeat(progressWidth - filled);

  return (
    <Box flexDirection="column">
      <Box justifyContent="space-between">
        <Text bold color="cyan">
          bc Setup Wizard
        </Text>
        <Text dimColor>~2 min</Text>
      </Box>

      <Box marginTop={0}>
        <Text dimColor>
          Step {currentStep + 1}/{totalSteps}: {stepTitle}
        </Text>
      </Box>

      <Box marginTop={0}>
        <Text color="green">{progressBar}</Text>
      </Box>
    </Box>
  );
}

interface StepContentProps {
  step: string;
  selectedOption: number;
  config: WizardConfig;
  isNarrow: boolean;
}

function StepContent({
  step,
  selectedOption,
  config,
  isNarrow,
}: StepContentProps): React.ReactElement {
  switch (step) {
    case 'welcome':
      return (
        <Box flexDirection="column">
          <Text>Welcome to bc - AI Agent Orchestration</Text>
          <Text dimColor>
            {isNarrow
              ? "Let's set up your workspace."
              : "This wizard will help you set up your first workspace."}
          </Text>
          <Box marginTop={1} flexDirection="column">
            <OptionItem index={0} selected={selectedOption === 0} label="Continue setup" />
            <OptionItem index={1} selected={selectedOption === 1} label="Skip (use defaults)" />
          </Box>
        </Box>
      );

    case 'preset':
      return (
        <Box flexDirection="column">
          <Text>Choose your team structure:</Text>
          <Box marginTop={1} flexDirection="column">
            <OptionItem
              index={0}
              selected={selectedOption === 0}
              label="Solo"
              desc="1 engineer agent"
            />
            <OptionItem
              index={1}
              selected={selectedOption === 1}
              label="Pair"
              desc="2 engineer agents"
            />
            <OptionItem
              index={2}
              selected={selectedOption === 2}
              label="Team"
              desc="PM + 2 engineers + UX"
            />
            <OptionItem
              index={3}
              selected={selectedOption === 3}
              label="Custom"
              desc="Configure manually"
            />
          </Box>
        </Box>
      );

    case 'agents':
      return (
        <Box flexDirection="column">
          <Text>How many agents to create?</Text>
          <Box marginTop={1} flexDirection="column">
            <OptionItem index={0} selected={selectedOption === 0} label="1 agent" />
            <OptionItem index={1} selected={selectedOption === 1} label="2 agents" />
            <OptionItem index={2} selected={selectedOption === 2} label="3 agents" />
            <OptionItem index={3} selected={selectedOption === 3} label="4+ agents" />
          </Box>
        </Box>
      );

    case 'channels':
      return (
        <Box flexDirection="column">
          <Text>Default channels to create:</Text>
          <Box marginTop={1} flexDirection="column">
            <OptionItem
              index={0}
              selected={selectedOption === 0}
              label="Basic (2)"
              desc="#general, #engineering"
            />
            <OptionItem
              index={1}
              selected={selectedOption === 1}
              label="Standard (3)"
              desc="+ #design"
            />
            <OptionItem
              index={2}
              selected={selectedOption === 2}
              label="Full (4)"
              desc="+ #product"
            />
          </Box>
        </Box>
      );

    case 'complete':
      return (
        <Box flexDirection="column">
          <Text bold color="green">
            Setup Complete!
          </Text>
          <Box marginTop={1} flexDirection="column">
            <Text>Your workspace is ready with:</Text>
            <Text>
              • Preset: <Text color="cyan">{config.preset}</Text>
            </Text>
            <Text>
              • Agents: <Text color="cyan">{config.agentCount}</Text>
            </Text>
            <Text>
              • Channels: <Text color="cyan">{config.channelCount}</Text>
            </Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            <OptionItem index={0} selected={selectedOption === 0} label="Finish and start" />
            <OptionItem index={1} selected={selectedOption === 1} label="Start over" />
          </Box>
        </Box>
      );

    default:
      return <Text>Unknown step</Text>;
  }
}

interface OptionItemProps {
  index: number;
  selected: boolean;
  label: string;
  desc?: string;
}

function OptionItem({ selected, label, desc }: OptionItemProps): React.ReactElement {
  return (
    <Box>
      <Text color={selected ? 'cyan' : undefined}>
        {selected ? '▸ ' : '  '}
        {label}
      </Text>
      {desc && <Text dimColor> - {desc}</Text>}
    </Box>
  );
}

interface WizardFooterProps {
  isNarrow: boolean;
}

function WizardFooter({ isNarrow }: WizardFooterProps): React.ReactElement {
  const hints = isNarrow
    ? 'j/k:nav Enter:select Esc:cancel'
    : 'j/k: navigate | Enter: select | s: skip | b: back | Esc: cancel';

  return (
    <Box borderStyle="single" borderColor="gray" paddingX={1}>
      <Text dimColor>{hints}</Text>
    </Box>
  );
}

export default SetupWizard;
