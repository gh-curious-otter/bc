# Frontend Engineering Review

**Date:** 2026-03-21 (rev 2 — re-audit after PR #2153 merged)
**Repo:** gh-curious-otter/bc
**Maturity Score: 6/10**

## Context

bc is a CLI-first AI agent orchestrator. Three React frontends:
- **TUI** (`tui/`) — Ink 4 terminal UI, ~100 source files, 111 test files, Bun runtime. The primary interface.
- **Web** (`web/`) — Vite + Tailwind + react-router-dom SPA, ~24 source files, zero tests. Dashboard served by bcd.
- **Landing** (`landing/`) — Next.js 16 static export, React 19, framer-motion, Playwright tests. Marketing site.

The project is pre-release and partially built by its own AI agents. The TUI is the most mature surface. The web dashboard is early-stage. The landing page is polished but has performance concerns.

## What Changed Since v1 Review

PR #2153 fixed 5 issues:
- ~~ConfigContext/UnreadContext memoization~~ -> FIXED (useMemo applied)
- ~~RootProvider inline object~~ -> FIXED (useMemo applied)
- ~~`@bc/tui` self-reference~~ -> FIXED (removed from package.json)
- ~~HOME regex escape~~ -> FIXED (special chars escaped)
- ~~`text-bc-fg` undefined class~~ -> FIXED (changed to `text-bc-text`)

Issue #2121 (ThemeToggle crash) was **not a bug** — `ThemeToggle` imports from the correct `_contexts/ThemeContext.tsx` which has `resolvedTheme`. The old `_components/ThemeProvider.tsx` is unused dead code.

---

## Open Bugs (functional, user-facing)

| # | Issue | Location | Status |
|---|-------|----------|--------|
| #2122 | Web API path injection — no `encodeURIComponent` | `web/src/api/client.ts:135` | OPEN |
| #2126 | Web missing 404 route — blank page on bad URL | `web/src/App.tsx` | OPEN |
| #2140 | Waitlist form `no-cors` swallows errors silently | `landing/src/app/waitlist/page.tsx:59` | OPEN |
| #2171 | Web Channels message duplication — WebSocket + fetch race | `web/src/views/Channels.tsx:67-104` | NEW |
| #2172 | Web Channels auto-scroll interrupts reading | `web/src/views/Channels.tsx:86-88` | NEW |
| NEW | TUI stale closure in AgentDetailView live mode | `tui/src/views/AgentDetailView.tsx:119` | NEW |
| NEW | TUI useListNavigation itemCount race on group collapse | `tui/src/hooks/useListNavigation.ts:103` | NEW |
| #2173 | TUI AgentsView setTimeout leak on unmount | `tui/src/views/AgentsView.tsx:195` | NEW |
| NEW | Web Layout "? help" hint is non-functional | `web/src/components/Layout.tsx:47` | NEW |

## Open Security Issues

| # | Issue | Location | Status |
|---|-------|----------|--------|
| #2122 | API parameter injection (no URL encoding) | `web/src/api/client.ts:135` | OPEN — **fix first** |
| #2125 | `_setSpawnForTesting` exported | `tui/src/services/bc.ts:75` | OPEN — superseded by #2155 (TUI->API) |

## Architecture Issues (still present)

| # | Issue | Location | Status |
|---|-------|----------|--------|
| #2124 | Landing page `"use client"` — entire page CSR | `landing/src/app/page.tsx:1` | OPEN |
| #2133 | AgentDetailView 595 lines | `tui/src/views/AgentDetailView.tsx` | OPEN |
| #2134 | CostsView 613 lines | `tui/src/views/CostsView.tsx` | OPEN |
| #2127 | Web: no code splitting (12 views eagerly imported) | `web/src/App.tsx:1-16` | OPEN |
| #2128 | Web: no AbortController on fetch | `web/src/api/client.ts` | OPEN |
| #2136 | TUI: useAgents debounce refs memory leak | `tui/src/hooks/useAgents.ts:74` | OPEN |
| #2135 | TUI: stale closure in useChannelsWithUnread | `tui/src/hooks/useChannels.ts:255` | OPEN |
| #2174 | TUI: bc.ts command cache unbounded (no LRU/max) | `tui/src/services/bc.ts:47` | NEW |
| NEW | Web: usePolling fetcher dep causes cascade re-renders | `web/src/hooks/usePolling.ts:22` | OPEN |
| NEW | Web: no request timeout or retry | `web/src/api/client.ts` | NEW |

## Design System Gap

Three disconnected color systems. Strategic direction: unify on **Solar Flare** palette from landing page.

| Token | Landing (Solar Flare) | Web (tokens.css) | TUI (themes.ts) |
|-------|----------------------|-------------------|------------------|
| Primary | `#EA580C` tangerine | `#60a5fa` blue | `'cyan'` |
| Background | `#0C0A08` warm black | `#0f1117` cool gray | terminal default |
| Surface | `#1E1A16` umber | `#1a1d27` cool slate | N/A |
| Border | `#2A2420` bark | `#2a2d3a` cool | `'gray'` |
| Accent | `#FB923C` amber | `#60a5fa` blue | `'magenta'` |

Tracked by: #2154 (design system epic), #2157 (shared tokens), #2158 (web migration), #2159 (TUI alignment)

## TUI->API Migration

The TUI spawns `bc` CLI subprocesses for data. The bcd server already has REST endpoints for everything. Migration tracked by #2155/#2160.

| Current | Target |
|---------|--------|
| `spawn('bc', ['status', '--json'])` | `fetch('/api/agents')` |
| ~100ms process startup per call | ~1ms fetch |
| In-memory cache in bc.ts | HTTP caching or SWR |
| Polling with setInterval | SSE via `/api/events` |
| `_setSpawnForTesting` attack surface | Standard fetch mocking |

---

## What's Solid

**TUI:** 111 test files, strict TS (zero `any`), error boundaries on every view, adaptive polling with backoff, command caching with TTLs, k9s-style navigation, configurable poll intervals via workspace config.

**Web:** Clean API client, ErrorBoundary per route, SSE via EventSource, design tokens via CSS vars, proper loading/error states in most views.

**Landing:** Excellent SEO (OG, Twitter, structured data, canonical), `prefers-reduced-motion` honored, Playwright E2E tests, responsive mobile-first, proper aria-hidden on decorative icons.

---

## Accessibility

**TUI (6/10):** j/k/g/G navigation works. Missing PageUp/PageDown (#2130). Focus trap incomplete for overlays (#2131). Light theme gray-on-white fails 4.5:1 contrast (#2132).

**Web (3/10):** No focus ring on nav (#2144). No skip-to-content. Table rows clickable without keyboard/ARIA support. Input fields missing labels. No a11y lint plugin.

**Landing (7/10):** Good aria usage, proper reduced-motion. Mobile menu doesn't trap focus. Form could improve.

---

## Prioritized Action Plan

### Immediate — functional bugs
1. **#2122** URL-encode web API path segments (security, 30 min)
2. **#2126** Add 404 route to web dashboard (10 min)
3. **#2171** Fix web Channels message deduplication (WebSocket + fetch race)
4. **#2172** Fix web Channels auto-scroll (only when at bottom)
5. **#2140** Fix waitlist form error handling

### Week 1 — architecture
6. **#2160** Create TUI API client (replace spawn with fetch) — biggest win
7. **#2157** Create shared design tokens package (Solar Flare)
8. **#2127** Add React.lazy to web routes
9. **#2128** Add AbortController to web API client
10. Fix TUI stale closure in AgentDetailView live mode

### Week 2 — polish
11. **#2158** Migrate web to Solar Flare palette
12. **#2159** Align TUI theme with Solar Flare
13. **#2137** Migrate 40+ hardcoded TUI colors (do with #2159)
14. **#2130** Add PageUp/PageDown to TUI lists
15. **#2144** Add focus indicators to web nav

### Week 3 — testing & cleanup
16. **#2138** Set up web test framework (Vitest)
17. **#2139** Migrate TUI skipped Jest tests to bun:test
18. **#2133/#2134** Split god components
19. **#2142** Fix ESLint version mismatch
20. **#2148** Delete unused `_components/ThemeProvider.tsx`

---

## Issue Tracker

### Closed (fixed)
- ~~#2121~~ ThemeToggle crash — not a bug (correct context used)
- ~~#2123~~ Context memoization — fixed in PR #2153
- ~~#2141~~ Self-reference dep — fixed in PR #2153
- ~~#2143~~ RootProvider inline object — fixed in PR #2153
- ~~#2145~~ HOME regex escape — fixed in PR #2153
- ~~#2149~~ text-bc-fg class — fixed in PR #2153

### Open epics
- #2154 Unified Design System (Solar Flare)
- #2155 TUI->API migration
- #2156 CLI->bcd routing
- #2111 Performance
- #2112 Security
- #2113 Accessibility
- #2114 Component architecture
- #2115 State management
- #2117 Testing
- #2118 Bundle optimization
- #2119 Error handling

### Master tracker: #2151
