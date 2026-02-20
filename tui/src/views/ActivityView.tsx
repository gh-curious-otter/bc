/**
 * ActivityView - Timeline view of agent activity and cost trends
 * Issue #1047: Add activity timeline and cost trend tracking
 *
 * Displays chronological view of agent activity with cost trends for the selected period.
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useActivityData } from '../hooks/useActivityData';
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
 * Activity Timeline View - shows agent activity and cost trends
 */
export function ActivityView({ disableInput = false }: ActivityViewProps): React.ReactElement {
  const [timePeriod, setTimePeriod] = useState<TimePeriod>('24h');
  const { isWide } = useResponsiveLayout();

  const { activities, loading: activitiesLoading, error: activitiesError, refresh: refreshActivities } = useActivityData({
    hours: timePeriod === '24h' ? 24 : timePeriod === 'week' ? 168 : 720,
  });

  const { budgetStatus, loading: trendLoading } = useCostTrends({
    period: timePeriod === '24h' ? 'day' : timePeriod === 'week' ? 'week' : 'month',
  });

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
    },
    { isActive: !disableInput }
  );

  const loading = activitiesLoading || trendLoading;

  if (loading) {
    return <LoadingIndicator message="Loading activity timeline..." />;
  }

  if (activitiesError) {
    return <ErrorDisplay error={activitiesError} onRetry={() => { void refreshActivities(); }} />;
  }

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
      </Box>

      {/* Cost Trend Summary */}
      <Box marginBottom={1} flexDirection="column" borderStyle="single" borderColor={STATUS_COLORS.info} paddingX={1}>
        <Text bold>Cost Summary</Text>
        <Box>
          <Text>Spent: ${budgetStatus.spent.toFixed(2)} / ${budgetStatus.budget.toFixed(2)}</Text>
          <Text dimColor> | </Text>
          <Text color={budgetStatus.status === 'critical' ? STATUS_COLORS.error : budgetStatus.status === 'warning' ? STATUS_COLORS.warning : STATUS_COLORS.info}>
            {budgetStatus.percentUsed}% used
          </Text>
        </Box>
        <Box>
          <Text dimColor>Burn rate: ${budgetStatus.burnRate.toFixed(2)}/day | Projected: ${budgetStatus.projectedTotal.toFixed(2)}</Text>
        </Box>
      </Box>

      {/* Activity Timeline */}
      <Box flexDirection="column" marginBottom={1} borderStyle="single" borderColor={STATUS_COLORS.working} paddingX={1}>
        <Text bold>Agent Activity</Text>
        {activities.length === 0 ? (
          <Text dimColor>No activity recorded in this period</Text>
        ) : (
          activities.slice(0, isWide ? 15 : 8).map((activity, idx) => (
            <Box key={idx}>
              <Text dimColor>{activity.startTime.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })}</Text>
              <Text dimColor>-</Text>
              <Text>{activity.agents.join(', ')}</Text>
              <Text dimColor> ({activity.duration}m)</Text>
            </Box>
          ))
        )}
      </Box>

      {/* Footer */}
      <Footer
        hints={[
          { key: 'd', label: '24h' },
          { key: 'w', label: 'week' },
          { key: 'm', label: 'month' },
        ]}
      />
    </Box>
  );
}

export default ActivityView;
