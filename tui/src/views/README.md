# Views Directory Structure

This document describes the standard organization pattern for TUI view components.

## Issue #1605

## Overview

Views are organized in two patterns depending on complexity:

### Simple Views (< 200 lines)
Single file in `views/` directory:
```
views/
├── SimpleView.tsx        # Self-contained view
```

### Complex Views (> 200 lines)
Directory with extracted components:
```
views/
├── ComplexView/
│   ├── index.ts              # Re-exports main view
│   ├── ComplexView.tsx       # Container/orchestrator
│   └── components/           # Extracted sub-components
│       ├── FeatureA.tsx
│       └── FeatureB.tsx
```

## Current Structure

### Already Following Pattern
- `agents/` - AgentCard, AgentList, AgentActions, etc.
- `components/channels/` - ChannelRow, ChannelHistoryView

### Single File Views (appropriate size)
- `ActivityView.tsx` (~130 lines) - Activity timeline
- `DemonsView.tsx` (~350 lines) - Could be split
- `ProcessesView.tsx` (~200 lines)
- `RoutingView.tsx` (~290 lines)
- `RolesView.tsx` (~460 lines) - Could be split

### Views That Could Be Refactored
Large views that would benefit from extraction:
- `AgentsView.tsx` (643 lines) - Has agents/ but container is large
- `FilesView.tsx` (583 lines) - Would benefit from split
- `Dashboard.tsx` (529 lines) - Would benefit from split

## Guidelines

### When to Extract Components
1. Component is > 100 lines
2. Component has distinct responsibility
3. Component could be reused
4. Testing would be easier with isolation

### Naming Conventions
- View directories: PascalCase (`AgentsView/`)
- Component files: PascalCase (`AgentCard.tsx`)
- Index files: lowercase (`index.ts`)
- Hooks: camelCase with `use` prefix (`useAgentState.ts`)

### Export Pattern
```typescript
// views/ViewName/index.ts
export { ViewName } from './ViewName';
export { SubComponent } from './components/SubComponent';
```

```typescript
// views/index.ts (main barrel file)
export { ViewName } from './ViewName';  // Directory export
export { SimpleView } from './SimpleView';  // Direct file export
```

## Migration Steps

When refactoring a large view:

1. Create directory: `views/[ViewName]/`
2. Move main file: `[ViewName].tsx` -> `[ViewName]/[ViewName].tsx`
3. Create `index.ts` re-exporting the main view
4. Extract components to `[ViewName]/components/`
5. Update imports in the main view
6. Update `views/index.ts` to use directory export
