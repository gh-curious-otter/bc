/**
 * ActivityView - Timeline view of agent activity and cost trends
 * Issue #1047: Add activity timeline and cost trend tracking
 *
 * Displays chronological view of agent activity with cost trends for the selected period.
 */

import React, { useState, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { useActivityData, type ActivityPeriod } from '../hooks/useActivityData';
import { useCostTrends } from '../hooks/useCostTrends';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout';
import { Footer } from '../components/Footer';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { STATUS_COLORS } from '../theme/StatusColors';

type TimePeriod = '24h' | 'week' | 'month';

interface ActivityViewProps {
  disableInput?: boolean;
}

/**
 * Generate a simple progress bar for budget visualization
 */
function ProgressBar({ percent, width = 20 }: { percent: number; width?: number }): React.ReactElement {
  const filled = Math.min(Math.round((percent / 100) * width), width);
  const empty = width - filled;
  const bar = '█'.repeat(filled) + '░'.repeat(empty);

  let color: string = STATUS_COLORS.info;
  if (percent >= 90) {
    color = STATUS_COLORS.error;
  } else if (percent >= 70) {
    color = STATUS_COLORS.warning;
  }

  return <Text color={color}>{bar}</Text>;
}

/**
 * Get activity indicator symbol based on event count
 */
function getActivityIndicator(eventCount: number): string {
  if (eventCount >= 10) return '⊙⊙⊙';
  if (eventCount >= 5) return '⊙⊙';
  if (eventCount >= 1) return '⊙';
  return '○';
}

/**
 * Summarize agent activity counts
 */
function summarizeAgentActivity(activities: ActivityPeriod[]): Map<string, number> {
  const summary = new Map<string, number>();
  for (const activity of activities) {
    for (const agent of activity.agents) {
      summary.set(agent, (summary.get(agent) ?? 0) + activity.eventCount);
    }
  }
  return summary;
}

/**
 * Activity Timeline View - shows agent activity and cost trends
 */
export function ActivityView({ disableInput = false }: ActivityViewProps): React.ReactElement {
  const [timePeriod, setTimePeriod] = useState<TimePeriod>('24h');
  const { isWide } = useResponsiveLayout();

  const { activities, loading: activitiesLoading, error: activitiesError, refresh } = useActivityData({
    hours: timePeriod === '24h' ? 24 : timePeriod === 'week' ? 168 : 720,
  });

  const { budgetStatus, trends, loading: trendLoading } = useCostTrends({
    period: timePeriod === '24h' ? 'day' : timePeriod === 'week' ? 'week' : 'month',
  });

  // Summarize agent activity
  const agentSummary = useMemo(() => summarizeAgentActivity(activities), [activities]);
  const sortedAgents = useMemo(
    () => Array.from(agentSummary.entries()).sort((a, b) => b[1] - a[1]),
    [agentSummary]
  );

  // Keyboard navigation
  useInput(
    (input) => {
      if (input === 'd') {
        setTimePeriod('24h');
      }
      if (input === 'w') {
        setTimePeriod('week');
      }
      if (input === 'm') {
        setTimePeriod('month');
      }
      if (input === 'r') {
        void refresh();
      }
    },
    { isActive: !disableInput }
  );

  const loading = activitiesLoading || trendLoading;

  if (loading) {
    return <LoadingIndicator message="Loading activity timeline..." />;
  }

  if (activitiesError) {
    return <ErrorDisplay error={activitiesError} onRetry={() => { void refresh(); }} />;
  }

  const maxRows = isWide ? 12 : 6;

  return (
    <Box flexDirection="column" width="100%">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color={STATUS_COLORS.info}>
          Activity Timeline
        </Text>
        <Text dimColor> - {timePeriod === '24h' ? 'Last 24 Hours' : timePeriod === 'week' ? 'Last 7 Days' : 'Last 30 Days'}</Text>
      </Box>

      {/* Time period selector */}
      <Box marginBottom={1}>
        <Text dimColor>Period: </Text>
        <Text color={timePeriod === '24h' ? STATUS_COLORS.working : 'white'}>[d] 24h</Text>
        <Text dimColor> | </Text>
        <Text color={timePeriod === 'week' ? STATUS_COLORS.working : 'white'}>[w] Week</Text>
        <Text dimColor> | </Text>
        <Text color={timePeriod === 'month' ? STATUS_COLORS.working : 'white'}>[m] Month</Text>
        <Text dimColor> | </Text>
        <Text dimColor>[r] Refresh</Text>
      </Box>

      {/* Cost & Budget Summary */}
      <Box marginBottom={1} flexDirection="column" borderStyle="single" borderColor={STATUS_COLORS.info} paddingX={1}>
        <Text bold>Cost Summary</Text>
        <Box>
          <Text>Spent: </Text>
          <Text color={budgetStatus.status === 'critical' ? STATUS_COLORS.error : STATUS_COLORS.info}>
            ${budgetStatus.spent.toFixed(2)}
          </Text>
          <Text> / ${budgetStatus.budget.toFixed(2)}</Text>
        </Box>
        <Box>
          <ProgressBar percent={budgetStatus.percentUsed} width={isWide ? 30 : 20} />
          <Text> </Text>
          <Text color={budgetStatus.status === 'critical' ? STATUS_COLORS.error : budgetStatus.status === 'warning' ? STATUS_COLORS.warning : STATUS_COLORS.info}>
            {budgetStatus.percentUsed}%
          </Text>
        </Box>
        <Box>
          <Text dimColor>Burn: ${budgetStatus.burnRate.toFixed(2)}/day</Text>
          <Text dimColor> │ </Text>
          <Text dimColor>Projected: </Text>
          <Text color={budgetStatus.projectedTotal > budgetStatus.budget ? STATUS_COLORS.warning : 'white'}>
            ${budgetStatus.projectedTotal.toFixed(2)}
          </Text>
          <Text dimColor> │ </Text>
          <Text dimColor>{budgetStatus.daysRemaining}d left</Text>
        </Box>
      </Box>

      {/* Agent Activity Summary */}
      {sortedAgents.length > 0 && (
        <Box marginBottom={1} flexDirection="column" borderStyle="single" borderColor={STATUS_COLORS.working} paddingX={1}>
          <Text bold>Agent Activity ({timePeriod})</Text>
          {sortedAgents.slice(0, isWide ? 6 : 4).map(([agent, count]) => (
            <Box key={agent}>
              <Text dimColor>{getActivityIndicator(count)} </Text>
              <Text color={STATUS_COLORS.working}>{agent}</Text>
              <Text dimColor>: {count} events</Text>
            </Box>
          ))}
        </Box>
      )}

      {/* Activity Timeline */}
      <Box flexDirection="column" marginBottom={1} borderStyle="single" borderColor="gray" paddingX={1}>
        <Text bold>Timeline</Text>
        {activities.length === 0 ? (
          <Text dimColor>No activity recorded in this period</Text>
        ) : (
          activities.slice(0, maxRows).map((activity, idx) => (
            <Box key={idx}>
              <Text dimColor>
                {activity.startTime.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })}
              </Text>
              <Text dimColor> │ </Text>
              <Text dimColor>{getActivityIndicator(activity.eventCount)} </Text>
              <Text>{activity.agents.slice(0, isWide ? 5 : 3).join(', ')}</Text>
              {activity.agents.length > (isWide ? 5 : 3) && <Text dimColor> +{activity.agents.length - (isWide ? 5 : 3)}</Text>}
              <Text dimColor> ({activity.eventCount} events)</Text>
            </Box>
          ))
        )}
        {activities.length > maxRows && (
          <Text dimColor>... {activities.length - maxRows} more periods</Text>
        )}
      </Box>

      {/* Cost by Agent (if available) */}
      {trends.length > 0 && isWide && (
        <Box marginBottom={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1}>
          <Text bold>Cost by Agent</Text>
          {trends.slice(0, 5).map((trend, idx) => (
            <Box key={idx}>
              <Text>{trend.period}</Text>
              <Text dimColor>: </Text>
              <Text color={STATUS_COLORS.info}>${trend.totalCost.toFixed(2)}</Text>
            </Box>
          ))}
        </Box>
      )}

      {/* Footer */}
      <Footer
        hints={[
          { key: 'd', label: '24h' },
          { key: 'w', label: 'week' },
          { key: 'm', label: 'month' },
          { key: 'r', label: 'refresh' },
        ]}
      />
    </Box>
  );
}

export default ActivityView;
