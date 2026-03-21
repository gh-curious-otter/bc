# bc Architecture

This document describes the core architecture of bc, the multi-agent orchestration CLI.

## Overview

bc enables coordinated AI agent teams with role-based hierarchy, channel communication, and isolated workspaces.

```mermaid
graph TB
    subgraph "bc CLI"
        CLI[bc command]
        TUI[Terminal UI]
    end

    subgraph "Agent Orchestration"
        ROOT[Root Agent]
        MGR[Manager Agents]
        ENG[Engineer Agents]
        UX[UX Agents]
    end

    subgraph "Communication"
        CH[Channels]
        MEM[Memory]
    end

    subgraph "Infrastructure"
        TMUX[tmux Sessions]
        WT[Git Worktrees]
        DB[SQLite State]
    end

    CLI --> ROOT
    CLI --> TUI
    ROOT --> MGR
    MGR --> ENG
    MGR --> UX
    ROOT --> CH
    MGR --> CH
    ENG --> CH
    UX --> CH
    ENG --> MEM
    ROOT --> TMUX
    MGR --> TMUX
    ENG --> TMUX
    ENG --> WT
    CH --> DB
    MEM --> DB
```

## Core Components

### 1. Agent Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Starting: bc agent create
    Starting --> Working: Initialized
    Working --> Idle: Task complete
    Working --> Stuck: No progress (30min)
    Idle --> Working: New task assigned
    Stuck --> Working: Manual intervention
    Working --> Stopped: bc agent stop
    Stopped --> Starting: bc agent start
    Stopped --> [*]: bc agent kill
```

### 2. Role Hierarchy

```mermaid
graph TD
    ROOT[root<br/>Full permissions]
    MGR[manager<br/>Create agents, assign work]
    ENG[engineer<br/>Implement tasks]
    PM[pm<br/>Planning, coordination]
    UX[ux<br/>Design, review]

    ROOT --> MGR
    ROOT --> PM
    MGR --> ENG
    MGR --> UX
    PM --> ENG

    style ROOT fill:#f96,stroke:#333
    style MGR fill:#9cf,stroke:#333
    style ENG fill:#9f9,stroke:#333
    style PM fill:#fc9,stroke:#333
    style UX fill:#f9f,stroke:#333
```

### 3. Channel Communication

```mermaid
sequenceDiagram
    participant M as Manager
    participant E1 as Engineer-01
    participant E2 as Engineer-02
    participant CH as #eng Channel

    M->>CH: "Implement feature X"
    CH-->>E1: Message received
    CH-->>E2: Message received
    E1->>CH: "Taking this task"
    E2->>CH: "I'll review"
    E1->>CH: "PR #123 ready"
    E2->>CH: "LGTM, approved"
```

### 4. Workspace Structure

```mermaid
graph TD
    subgraph ".bc/ Directory"
        CONFIG[config.toml]
        STATE[state.db]
        ROLES[roles/]
        AGENTS[agents/]
        WT[worktrees/]
    end

    subgraph "roles/"
        R_ROOT[root.md]
        R_ENG[engineer.md]
        R_MGR[manager.md]
    end

    subgraph "agents/"
        A1[eng-01/]
        A2[eng-02/]
    end

    subgraph "worktrees/"
        W1[eng-01/]
        W2[eng-02/]
    end

    ROLES --> R_ROOT
    ROLES --> R_ENG
    ROLES --> R_MGR
    AGENTS --> A1
    AGENTS --> A2
    WT --> W1
    WT --> W2
```

## Data Flow

### Message Storage (SQLite)

```mermaid
erDiagram
    CHANNELS ||--o{ MESSAGES : contains
    MESSAGES ||--o{ REACTIONS : has
    AGENTS ||--o{ MESSAGES : sends

    CHANNELS {
        string name PK
        timestamp created_at
    }

    MESSAGES {
        int id PK
        string channel FK
        string agent FK
        string content
        string type
        timestamp created_at
    }

    REACTIONS {
        int message_id FK
        string agent
        string emoji
    }

    AGENTS {
        string name PK
        string role
        string state
        timestamp created_at
    }
```

### Cost Tracking

```mermaid
graph LR
    subgraph "Agent Operations"
        A[Agent Task]
        API[API Call]
    end

    subgraph "Cost Tracking"
        LOG[Log Tokens]
        CALC[Calculate Cost]
        STORE[Store in DB]
    end

    subgraph "Reporting"
        BY_AGENT[By Agent]
        BY_MODEL[By Model]
        BY_TEAM[By Team]
    end

    A --> API
    API --> LOG
    LOG --> CALC
    CALC --> STORE
    STORE --> BY_AGENT
    STORE --> BY_MODEL
    STORE --> BY_TEAM
```

## TUI Architecture

```mermaid
graph TD
    subgraph "React/Ink App"
        APP[App.tsx]
        NAV[NavigationContext]
        FOCUS[FocusContext]
    end

    subgraph "Views"
        DASH[Dashboard]
        AGENTS[AgentsView]
        CHANNELS[ChannelsView]
        COSTS[CostsView]
    end

    subgraph "Components"
        DRAWER[Drawer]
        TABLE[Table]
        PANEL[Panel]
    end

    subgraph "Hooks"
        USE_AGENTS[useAgents]
        USE_RESP[useResponsiveLayout]
    end

    APP --> NAV
    APP --> FOCUS
    APP --> DRAWER
    NAV --> DASH
    NAV --> AGENTS
    NAV --> CHANNELS
    NAV --> COSTS
    DASH --> PANEL
    AGENTS --> TABLE
    USE_AGENTS --> AGENTS
    USE_RESP --> DRAWER
```

## See Also

- [CONTRIBUTING.md](../../CONTRIBUTING.md) - How to contribute
- [CLAUDE.md](../../.claude/CLAUDE.md) - Development guide
- [VISION.md](../../VISION.md) - Project vision
