/**
 * Views barrel export for bc TUI
 *
 * Issue #1605: Establish consistent view component structure
 *
 * Views are organized in two patterns:
 * - Simple views: Direct file export (./ViewName.tsx)
 * - Complex views: Directory export (./ViewName/index.ts)
 *
 * See README.md for the complete structure pattern.
 */

// Workspace views
export { Dashboard } from './Dashboard';
export { AgentsView } from './AgentsView';
export { AgentDetailView } from './AgentDetailView';
export { ChannelsView } from './ChannelsView';
export { FilesView } from './FilesView';
export { CommandsView } from './CommandsView';
export { CostsView } from './CostsView';

// Monitoring views
export { LogsView } from './LogsView';
export { ActivityView } from './ActivityView';
export { ProcessesView } from './ProcessesView';
export { DemonsView } from './DemonsView';

// System views
export { RolesView } from './RolesView';
export { TeamsView } from './TeamsView';
export { WorktreesView } from './WorktreesView';
export { WorkspaceSelectorView } from './WorkspaceSelectorView';
export { MemoryView } from './MemoryView';
export { RoutingView } from './RoutingView';

// Setup & utilities
export { SetupWizard } from './SetupWizard';

// Help
export { HelpView } from './HelpView';

// Re-export sub-components from complex views
export * from './agents';
