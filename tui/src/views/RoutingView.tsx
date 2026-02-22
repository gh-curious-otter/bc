/**
 * RoutingView - Display task routing configuration
 * Issue #1231 - Add additional TUI views
 */

import React, { useState, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import { useAgents } from '../hooks';

interface RoutingViewProps {
  onBack?: () => void;
  disableInput?: boolean;
}

// Static routing rules from pkg/routing/routing.go
const ROUTING_RULES = [
  {
    taskType: 'code',
    targetRole: 'engineer',
    description: 'Implementation work - coding features, bug fixes, refactoring',
    examples: ['Feature implementation', 'Bug fixes', 'Code refactoring'],
  },
  {
    taskType: 'review',
    targetRole: 'tech-lead',
    description: 'Code review work - PR reviews, architecture review',
    examples: ['PR review', 'Design review', 'Code quality checks'],
  },
  {
    taskType: 'merge',
    targetRole: 'manager',
    description: 'Merge operations - approved PRs integration',
    examples: ['Merge approved PRs', 'Release coordination'],
  },
  {
    taskType: 'qa',
    targetRole: 'qa',
    description: 'Quality assurance - testing and validation',
    examples: ['Test execution', 'Regression testing', 'UAT'],
  },
];

/**
 * RoutingView - Display and explain task routing rules
 */
export function RoutingView({
  onBack,
  disableInput = false,
}: RoutingViewProps): React.ReactElement {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetails, setShowDetails] = useState(false);
  const agents = useAgents();

  // Count agents by role
  const agentCountByRole = useMemo(() => {
    const counts: Record<string, number> = {};
    const agentList = agents.data ?? [];
    for (const agent of agentList) {
      counts[agent.role] = (counts[agent.role] || 0) + 1;
    }
    return counts;
  }, [agents.data]);

  // Count available (idle/working) agents by role
  const availableByRole = useMemo(() => {
    const counts: Record<string, number> = {};
    const agentList = agents.data ?? [];
    for (const agent of agentList) {
      if (agent.state === 'idle' || agent.state === 'working') {
        counts[agent.role] = (counts[agent.role] || 0) + 1;
      }
    }
    return counts;
  }, [agents.data]);

  const validIndex = Math.min(selectedIndex, ROUTING_RULES.length - 1);
  const currentRule = ROUTING_RULES[validIndex] as typeof ROUTING_RULES[0] | undefined;

  // Keyboard handling
  useInput(
    (input, key) => {
      if (showDetails) {
        if (key.escape || input === 'q' || key.return) {
          setShowDetails(false);
        }
        return;
      }

      if (key.upArrow || input === 'k') {
        setSelectedIndex(Math.max(0, validIndex - 1));
      } else if (key.downArrow || input === 'j') {
        setSelectedIndex(Math.min(ROUTING_RULES.length - 1, validIndex + 1));
      } else if (input === 'g') {
        setSelectedIndex(0);
      } else if (input === 'G') {
        setSelectedIndex(ROUTING_RULES.length - 1);
      } else if (key.return && currentRule !== undefined) {
        setShowDetails(true);
      } else if (input === 'q' || key.escape) {
        onBack?.();
      }
    },
    { isActive: !disableInput }
  );

  // Details view
  if (showDetails && currentRule !== undefined) {
    return (
      <Box flexDirection="column" padding={1}>
        <Panel title={`Route: ${currentRule.taskType}`} borderColor="cyan">
          <Box flexDirection="column">
            <Box marginBottom={1}>
              <Box width={15}>
                <Text dimColor>Task Type:</Text>
              </Box>
              <Text bold color="cyan">{currentRule.taskType}</Text>
            </Box>

            <Box marginBottom={1}>
              <Box width={15}>
                <Text dimColor>Target Role:</Text>
              </Box>
              <Text bold color="green">{currentRule.targetRole}</Text>
            </Box>

            <Box marginBottom={1}>
              <Box width={15}>
                <Text dimColor>Total Agents:</Text>
              </Box>
              <Text>{String(agentCountByRole[currentRule.targetRole] ?? 0)}</Text>
            </Box>

            <Box marginBottom={1}>
              <Box width={15}>
                <Text dimColor>Available:</Text>
              </Box>
              <Text color={availableByRole[currentRule.targetRole] ? 'green' : 'red'}>
                {String(availableByRole[currentRule.targetRole] ?? 0)}
              </Text>
            </Box>

            <Box marginBottom={1} flexDirection="column">
              <Text dimColor>Description:</Text>
              <Text wrap="wrap">{currentRule.description}</Text>
            </Box>

            <Box flexDirection="column">
              <Text dimColor>Example Tasks:</Text>
              <Box flexDirection="column" marginLeft={2}>
                {currentRule.examples.map((example, idx) => (
                  <Text key={idx} color="yellow">* {example}</Text>
                ))}
              </Box>
            </Box>
          </Box>
        </Panel>

        <Box marginTop={1}>
          <Text dimColor>[Enter/Esc/q] back to list</Text>
        </Box>
      </Box>
    );
  }

  // Main list view
  return (
    <ViewWrapper
      hints={[
        { key: 'j/k', label: 'navigate' },
        { key: 'Enter', label: 'details' },
      ]}
    >
      <Box flexDirection="column" width="100%">
        {/* Header with count (#1446) */}
        <HeaderBar
          title="Task Routing"
          count={ROUTING_RULES.length}
          subtitle="rules"
        />

        {/* Description */}
        <Box marginBottom={1} paddingX={1} borderStyle="single" borderColor="gray">
          <Text dimColor wrap="wrap">
            Task routing determines which agent role handles different types of work.
            The router uses round-robin selection among available agents of the target role.
          </Text>
        </Box>

        {/* Routing rules table */}
        <Panel title="Routing Rules">
          <Box flexDirection="column">
            {/* Header row */}
            <Box paddingX={1}>
              <Box width={12}>
                <Text bold dimColor>TASK TYPE</Text>
              </Box>
              <Box width={15}>
                <Text bold dimColor>TARGET ROLE</Text>
              </Box>
              <Box width={10}>
                <Text bold dimColor>AGENTS</Text>
              </Box>
              <Box width={12}>
                <Text bold dimColor>AVAILABLE</Text>
              </Box>
              <Box flexGrow={1}>
                <Text bold dimColor>DESCRIPTION</Text>
              </Box>
            </Box>

            {/* Rule rows */}
            {ROUTING_RULES.map((rule, idx) => (
              <RoutingRuleRow
                key={rule.taskType}
                rule={rule}
                selected={idx === validIndex}
                agentCount={agentCountByRole[rule.targetRole] ?? 0}
                availableCount={availableByRole[rule.targetRole] ?? 0}
              />
            ))}
          </Box>
        </Panel>

        {/* Agent Summary */}
        <Panel title="Role Summary">
          <Box flexDirection="row" flexWrap="wrap">
            {Object.entries(agentCountByRole)
              .sort((a, b) => b[1] - a[1])
              .map(([role, count]) => (
                <Box key={role} marginRight={3}>
                  <Text color="cyan">{role}: </Text>
                  <Text>{String(count)}</Text>
                  <Text dimColor> ({String(availableByRole[role] ?? 0)} avail)</Text>
                </Box>
              ))}
          </Box>
        </Panel>
      </Box>
    </ViewWrapper>
  );
}

interface RoutingRuleRowProps {
  rule: typeof ROUTING_RULES[0];
  selected: boolean;
  agentCount: number;
  availableCount: number;
}

function RoutingRuleRow({
  rule,
  selected,
  agentCount,
  availableCount,
}: RoutingRuleRowProps): React.ReactElement {
  const statusColor = availableCount > 0 ? 'green' : agentCount > 0 ? 'yellow' : 'red';

  return (
    <Box paddingX={1}>
      <Box width={12}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {selected ? '> ' : '  '}
          {rule.taskType}
        </Text>
      </Box>
      <Box width={15}>
        <Text color="green">{rule.targetRole}</Text>
      </Box>
      <Box width={10}>
        <Text>{String(agentCount)}</Text>
      </Box>
      <Box width={12}>
        <Text color={statusColor}>{String(availableCount)}</Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>{truncate(rule.description, 35)}</Text>
      </Box>
    </Box>
  );
}

/**
 * Truncate string to max length
 */
function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '...';
}

export default RoutingView;
