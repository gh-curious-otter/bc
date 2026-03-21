# Frontend Engineering Review

**Date:** 2026-03-20 (updated 2026-03-21)
**Repo:** gh-curious-otter/bc
**Stack:**
- **TUI**: React 18 + Ink 4 (terminal UI), TypeScript, Bun
- **Web**: React 18 + Vite 6 + Tailwind 3 + react-router-dom 6
- **Landing**: Next.js 16 + React 19 + Tailwind v4 + framer-motion + lucide-react

**Frontend Maturity Score: 6/10**

## Executive Summary

The bc project contains **three frontend surfaces**: a terminal UI (TUI), a web dashboard, and a marketing landing page. The TUI is the most mature — well-tested (111 test files), strict TypeScript, proper error boundaries, and solid architecture. However, all three frontends share common issues: **hardcoded colors bypassing the theme system**, **missing accessibility features**, and **no code splitting**. The web dashboard has **zero tests** and **API parameter injection**. The landing page renders entirely client-side despite being a static export, hurting First Contentful Paint.

**Strategic direction:** Unify all three frontends on the landing page's **Solar Flare** design system, migrate the TUI from CLI subprocess spawning to the bcd REST API, and route CLI commands through bcd when it's running.

---

## Component Architecture Map

```
bc/
+-- tui/src/                    # Terminal UI (Ink/React)
|   +-- app.tsx                 # Root: RootProvider > NavigationProvider > FocusProvider > ...
|   +-- views/                  # 12 views (Dashboard, Agents, Channels, Costs, Logs, Roles, ...)
|   |   +-- agents/             # Decomposed: AgentCard, AgentList, AgentPeekPanel, ...
|   |   +-- AgentDetailView.tsx # 595-line god component
|   |   +-- CostsView.tsx       # 613-line god component
|   +-- components/             # Shared: Table, Panel, ErrorBoundary, CommandBar, FilterBar, ...
|   +-- hooks/                  # useAgents, useChannels, usePolling, useFocusStateMachine, ...
|   +-- services/bc.ts          # CLI wrapper with caching (903 lines)
|   +-- theme/                  # ThemeContext, dark/light themes
|   +-- navigation/             # NavigationContext, FocusContext, TabBar
|   +-- config/                 # ConfigContext (performance tuning)
|
+-- web/src/                    # Web Dashboard (Vite/React)
|   +-- App.tsx                 # BrowserRouter with 12 routes, no lazy loading
|   +-- views/                  # Dashboard, Agents, Channels, Costs, Roles, Tools, MCP, ...
|   +-- components/             # Layout, ErrorBoundary, StatusBadge, Table
|   +-- hooks/                  # usePolling, useWebSocket
|   +-- api/client.ts           # REST client with type assertions
|   +-- theme/tokens.css        # CSS custom properties
|
+-- landing/src/app/            # Marketing Site (Next.js 16)
    +-- page.tsx                # 621-line "use client" landing page
    +-- _components/            # Nav, Footer, AnimatedBackground, TerminalComponents, ...
    +-- _contexts/              # ThemeContext (active); _components/ThemeProvider.tsx is dead code
    +-- docs/, waitlist/, product/, privacy/, terms/
```

---

## Critical Issues (user-facing impact)

| # | Issue | Location | Category | Impact | Status |
|---|-------|----------|----------|--------|--------|
| 1 | Landing page entirely client-rendered | `landing/src/app/page.tsx:1` | performance | FCP delayed 3-5s on slow connections | OPEN #2124 |
| 2 | AnimatedBackground O(n^2) particle loop | `landing/src/app/_components/AnimatedBackground.tsx` | performance | 6400 calcs/frame; battery drain | OPEN #2129 |
| ~~3~~ | ~~ThemeToggle uses undefined `resolvedTheme`~~ | ~~`landing/src/app/_components/ThemeToggle.tsx:7`~~ | ~~bug~~ | ~~Runtime crash~~ | **NOT A BUG** — uses correct `_contexts/ThemeContext` which has `resolvedTheme` |
| ~~4~~ | ~~ConfigContext value not memoized~~ | ~~`tui/src/config/ConfigContext.tsx:102`~~ | ~~performance~~ | ~~Cascade re-renders~~ | **FIXED** PR #2153 |
| ~~5~~ | ~~UnreadContext value not memoized~~ | ~~`tui/src/hooks/UnreadContext.tsx:115`~~ | ~~performance~~ | ~~Cascade re-renders~~ | **FIXED** PR #2153 |
| 6 | Web API parameter injection | `web/src/api/client.ts:135` | security | URL path injection via unsanitized names | OPEN #2122 |
| 7 | Web dashboard has zero tests | `web/` | testing | No regression detection | OPEN #2138 |
| 8 | TUI spawn function exported | `tui/src/services/bc.ts:75` | security | Can intercept CLI commands | OPEN #2125 — superseded by #2155 |
| 9 | Waitlist form silently fails | `landing/src/app/waitlist/page.tsx:59` | ux | `no-cors` swallows errors | OPEN #2140 |
| 10 | Web missing 404 route | `web/src/App.tsx` | ux | Blank page on invalid URLs | OPEN #2126 |
| **NEW** | Web Channels message duplication | `web/src/views/Channels.tsx:67-104` | bug | WebSocket + fetch race — messages appear twice | OPEN #2171 |
| **NEW** | Web Channels auto-scroll interrupts reading | `web/src/views/Channels.tsx:86-88` | ux | Scrolls to bottom on every message update | OPEN #2172 |
| **NEW** | TUI stale closure in AgentDetailView live mode | `tui/src/views/AgentDetailView.tsx:119` | bug | `isFollowing` captured at closure creation time | NEW |
| **NEW** | TUI AgentsView setTimeout leak on unmount | `tui/src/views/AgentsView.tsx:195` | bug | setState on unmounted component | OPEN #2173 |

## Major Issues (quality & maintainability)

| # | Issue | Location | Category | Impact | Status |
|---|-------|----------|----------|--------|--------|
| 11 | AgentDetailView god component (595 lines) | `tui/src/views/AgentDetailView.tsx` | components | 9 useState, all-in-one | OPEN #2133 |
| 12 | CostsView god component (613 lines) | `tui/src/views/CostsView.tsx` | components | Data + sorting + 2 layouts + detail | OPEN #2134 |
| 13 | 40+ hardcoded colors bypass theme | Multiple TUI files | styling | Views ignore dark/light switching | OPEN #2137 |
| 14 | Web: no code splitting | `web/src/App.tsx:1-16` | bundle | All 12 views eagerly imported | OPEN #2127 |
| 15 | Web: no AbortController | `web/src/api/client.ts` | data-fetching | Memory leaks on rapid view switching | OPEN #2128 |
| 16 | Web: usePolling race condition | `web/src/hooks/usePolling.ts:22` | data-fetching | Fetcher changes trigger constant polling | OPEN |
| 17 | TUI ESLint version mismatch | `tui/package.json:31-38` | dx | typescript-eslint v7 mixed with v6 | OPEN #2142 |
| ~~18~~ | ~~TUI self-reference in deps~~ | ~~`tui/package.json:20`~~ | ~~bundle~~ | ~~Circular dependency~~ | **FIXED** PR #2153 |
| 19 | Missing PageUp/PageDown navigation | `tui/src/hooks/useListNavigation.ts` | a11y | No fast navigation in long lists | OPEN #2130 |
| 20 | Focus trap incomplete for overlays | `tui/navigation/FocusContext.tsx` | a11y | Keys leak through overlays | OPEN #2131 |
| 21 | Light theme contrast insufficient | `tui/src/theme/themes.ts` | a11y | Gray on white < 4.5:1 ratio | OPEN #2132 |
| 22 | useAgents debounce refs never cleaned | `tui/src/hooks/useAgents.ts:74` | state-mgmt | Memory leak as agents churn | OPEN #2136 |
| 23 | Stale closure in useChannelsWithUnread | `tui/src/hooks/useChannels.ts:255` | state-mgmt | Expensive unread recalculation | OPEN #2135 |
| 24 | 44 skipped tests (Jest mock migration) | Multiple TUI test files | testing | 15-20% tests not executing | OPEN #2139 |
| 25 | framer-motion adds ~35KB gzipped | `landing/package.json` | bundle | Heavy for a marketing site | OPEN |
| 26 | Duplicate ThemeProvider implementations | `landing/src/app/_components/` vs `_contexts/` | components | Dead code — old one is unused | OPEN #2148 (downgraded to low) |
| **NEW** | TUI bc.ts command cache unbounded | `tui/src/services/bc.ts:47` | state-mgmt | No LRU eviction, grows forever | OPEN #2174 |
| **NEW** | Web: no request timeout or retry | `web/src/api/client.ts` | data-fetching | Requests hang if backend down | NEW |

## Minor Issues & Polish

| # | Issue | Location | Category | Impact | Status |
|---|-------|----------|----------|--------|--------|
| 27 | Duplicate ESLint config files | `tui/.eslintrc.cjs` + `tui/eslint.config.js` | dx | Conflicting lint rules | OPEN #2142 |
| 28 | Namespace re-exports in barrel files | `tui/src/constants/index.ts` | bundle | Prevents tree-shaking | OPEN |
| 29 | WebVitals only logs in development | `landing/src/app/_components/WebVitals.tsx` | performance | No production CWV monitoring | OPEN #2146 |
| 30 | Web: hardcoded polling intervals | Multiple web views | data-fetching | Inconsistent refresh rates | OPEN |
| 31 | Web: missing dynamic page titles | `web/index.html` | seo | All tabs show same title | OPEN #2150 |
| ~~32~~ | ~~Web: `text-bc-fg/80` undefined class~~ | ~~`web/src/views/Roles.tsx:65`~~ | ~~styling~~ | ~~Falls back to browser default~~ | **FIXED** PR #2153 |
| 33 | Web: no focus ring on nav links | `web/src/components/Layout.tsx` | a11y | Keyboard users can't see focus | OPEN #2144 |
| 34 | Landing: redundant aria-labels | `landing/src/app/page.tsx` | a11y | Screen readers hear duplicate | OPEN |
| 35 | Table.tsx array index as key | `tui/src/components/Table.tsx:66` | performance | Reconciliation issues on reorder | OPEN |
| ~~36~~ | ~~HOME env var used in regex unescaped~~ | ~~`tui/src/components/ActivityFeed.tsx:66`~~ | ~~security~~ | ~~Regex injection~~ | **FIXED** PR #2153 |
| ~~37~~ | ~~RootProvider creates inline object~~ | ~~`tui/src/providers/RootProvider.tsx:35`~~ | ~~performance~~ | ~~Config recreated each render~~ | **FIXED** PR #2153 |
| 38 | FilterBar double state update | `tui/src/components/FilterBar.tsx:35` | performance | Two renders per keystroke | OPEN |
| **NEW** | Web Layout "? help" hint non-functional | `web/src/components/Layout.tsx:47` | ux | Hint shown but no handler | NEW |

---

## What's Done Well

**TUI:**
- Excellent test coverage (111 test files, 10,311 lines of test code, no snapshot tests)
- Strict TypeScript (`strict: true`, zero `any` in production code)
- Proper error boundaries wrapping every view
- Sophisticated polling with configurable intervals, debounce, change detection
- Command result caching with TTLs (stale-while-revalidate)
- Good focus management architecture (FocusContext + useFocusStateMachine)
- Proper process timeout handling with SIGTERM/SIGKILL escalation
- Adaptive polling that backs off when idle
- Well-organized constants (dimensions, timings, cache, limits)

**Web:**
- Clean API client with centralized fetch
- Proper ErrorBoundary on every route
- SSE via EventSource for real-time updates
- Design tokens via CSS custom properties
- Consistent Tailwind usage with custom theme

**Landing:**
- Excellent SEO metadata (OG, Twitter, structured data)
- Proper `prefers-reduced-motion` handling for animations
- Responsive design with mobile-first breakpoints
- Playwright E2E tests configured
- Good aria-hidden usage on decorative icons
- Static export for CDN hosting

---

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

## TUI to API Migration

The TUI spawns `bc` CLI subprocesses for data. The bcd server already has REST endpoints for everything. Migration tracked by #2155/#2160.

| Current | Target |
|---------|--------|
| `spawn('bc', ['status', '--json'])` | `fetch('/api/agents')` |
| ~100ms process startup per call | ~1ms fetch |
| In-memory cache in bc.ts | HTTP caching or SWR |
| Polling with setInterval | SSE via `/api/events` |
| `_setSpawnForTesting` attack surface | Standard fetch mocking |

---

## Bundle Analysis

| App | Estimated Size | Issues |
|-----|---------------|--------|
| TUI | ~3,200 source lines + 3 deps | ESLint version mismatch |
| Web | ~2,400 source lines, no code splitting | All 12 views in initial bundle |
| Landing | ~4,000 source lines + framer-motion (~35KB gz) | Client-rendered page, heavy particle animation |

---

## Accessibility Assessment

**TUI (6/10):** Keyboard navigation works via j/k/arrows but missing PageUp/PageDown. Focus trapping incomplete for overlays. Light theme has poor contrast. No terminal size warning for small screens.

**Web (3/10):** No focus indicators on navigation. No skip-to-content. No ARIA landmarks. No keyboard shortcuts. Missing loading states for async operations.

**Landing (7/10):** Good aria-hidden on icons, proper aria-labels on CTAs, prefers-reduced-motion honored. Minor redundant labels. Form accessibility could improve.

---

## Performance Assessment

**TUI:** FilterBar double-updates on each keystroke. Table uses array index keys. Agent debounce refs leak memory. Command cache grows unbounded. Overall solid polling architecture.

**Web:** No code splitting means full bundle on first load. No AbortController causes memory leaks. usePolling can race-condition on fetcher change. Channels view has message duplication from WebSocket + fetch race.

**Landing:** Entire page client-rendered with "use client". 80-particle O(n^2) animation on every frame. framer-motion adds ~35KB. Fonts loaded as render-blocking stylesheet.

---

## Action Plan

### Phase 1: Critical UX, Security & Bugs (immediate)
- Fix API parameter injection (web) #2122
- Add 404 route (web) #2126
- Fix Channels message duplication (web) #2171
- Fix Channels auto-scroll interrupting reading (web) #2172
- Fix waitlist form error handling (landing) #2140

### Phase 2: Architecture (week 1)
- Create TUI API client — replace spawn with fetch #2160
- Create shared design tokens package (Solar Flare) #2157
- Add React.lazy code splitting to web routes #2127
- Add AbortController to web API client #2128
- Convert landing page to Server Components #2124

### Phase 3: Design System & Accessibility (week 2)
- Migrate web to Solar Flare palette #2158
- Align TUI theme with Solar Flare #2159
- Migrate 40+ hardcoded TUI colors (do with #2159) #2137
- Add PageUp/PageDown to TUI lists #2130
- Add focus indicators to web navigation #2144

### Phase 4: Testing & Cleanup (week 3)
- Set up web test framework (Vitest) #2138
- Migrate TUI skipped Jest tests to bun:test #2139
- Split AgentDetailView / CostsView god components #2133 #2134
- Fix ESLint version mismatch #2142
- Delete unused `_components/ThemeProvider.tsx` #2148

---

## GitHub Issues Created

### Closed (fixed or not a bug)
- ~~#2121~~ ThemeToggle crash — not a bug (uses correct context)
- ~~#2123~~ Context memoization — fixed in PR #2153
- ~~#2141~~ Self-reference dep — fixed in PR #2153
- ~~#2143~~ RootProvider inline object — fixed in PR #2153
- ~~#2145~~ HOME regex escape — fixed in PR #2153
- ~~#2149~~ text-bc-fg class — fixed in PR #2153

### Open epics
- #2154 Unified Design System (Solar Flare)
- #2155 TUI to API migration
- #2156 CLI to bcd routing
- #2111 Performance & Core Web Vitals
- #2112 Frontend Security Hardening
- #2113 Accessibility Compliance
- #2114 Component Architecture Cleanup
- #2115 State Management & Data Fetching
- #2117 Frontend Testing
- #2118 Bundle Optimization & Code Splitting
- #2119 Error Handling & Edge Cases
- #2120 Frontend DX & Tooling

### Master tracker: #2151
