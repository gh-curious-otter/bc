# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-agent orchestration with role-based hierarchy
- TUI dashboard with 14 views (Dashboard, Agents, Channels, Costs, Commands, Roles, Logs, Worktrees, Workspaces, Demons, Processes, Memory, Routing, Help)
- Agent communication via channels with SQLite persistence
- Cost tracking with budgets and spending analytics
- Git worktree isolation for each agent
- Role system (root, manager, engineer, etc.) with capabilities
- Memory system for agent learnings and experiences
- Responsive TUI layout adapting to terminal size
- Command favorites with persistence
- Agent peek functionality for output inspection

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- TUI text corruption at various terminal widths
- Dashboard layout issues at narrow terminals
- Commands view character loss bug

### Security
- Input validation for all command handlers
- Environment variable key validation to prevent shell injection
