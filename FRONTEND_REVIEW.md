# Frontend Engineering Review

**Date:** 2026-03-20
**Repo:** gh-curious-otter/bc
**Stack:**
- **TUI**: React 18 + Ink 4 (terminal UI), TypeScript, Bun
- **Web**: React 18 + Vite 6 + Tailwind 3 + react-router-dom 6
- **Landing**: Next.js 16 + React 19 + Tailwind v4 + framer-motion + lucide-react

**Frontend Maturity Score: 6/10**

## Executive Summary

The bc project contains **three frontend surfaces**: a terminal UI (TUI), a web dashboard, and a marketing landing page. The TUI is the most mature — well-tested (111 test files), strict TypeScript, proper error boundaries, and solid architecture. However, all three frontends share common issues: **unmemoized context values causing cascade re-renders**, **hardcoded colors bypassing the theme system**, **missing accessibility features**, and **no code splitting**. The web dashboard has **zero tests** and **XSS-adjacent API parameter injection**. The landing page renders entirely client-side despite being a static export, killing First Contentful Paint. Security-wise, the TUI's spawn function injection is exported and the web API client lacks AbortController support.

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
    +-- _contexts/              # ThemeContext (duplicated with _components/ThemeProvider)
    +-- docs/, waitlist/, product/, privacy/, terms/
```

---

## Critical Issues (user-facing impact)

| # | Issue | Location | Category | Impact |
|---|-------|----------|----------|--------|
| 1 | Landing page entirely client-rendered | `landing/src/app/page.tsx:1` | performance | FCP delayed 3-5s on slow connections; poor SEO crawl efficiency |
| 2 | AnimatedBackground O(n^2) particle loop | `landing/src/app/_components/AnimatedBackground.tsx` | performance | 80 particles x 80 checks = 6400 calcs/frame; battery drain on mobile |
| 3 | ThemeToggle uses undefined `resolvedTheme` | `landing/src/app/_components/ThemeToggle.tsx:7` | bug | Runtime crash — theme toggle button broken |
| 4 | ConfigContext value not memoized | `tui/src/config/ConfigContext.tsx:102` | performance | All config consumers re-render on every provider render |
| 5 | UnreadContext value not memoized | `tui/src/hooks/UnreadContext.tsx:115` | performance | All unread consumers re-render unnecessarily |
| 6 | Web API parameter injection | `web/src/api/client.ts:135` | security | URL path injection via unsanitized channel names |
| 7 | Web dashboard has zero tests | `web/` | testing | No regression detection for web dashboard |
| 8 | TUI spawn function exported | `tui/src/services/bc.ts:75` | security | `_setSpawnForTesting` can intercept all CLI commands |
| 9 | Waitlist form silently fails | `landing/src/app/waitlist/page.tsx:59` | ux | `no-cors` fetch swallows errors; users think submission worked |
| 10 | Web missing 404 route | `web/src/App.tsx` | ux | Invalid URLs render blank page |

## Major Issues (quality & maintainability)

| # | Issue | Location | Category | Impact |
|---|-------|----------|----------|--------|
| 11 | AgentDetailView god component (595 lines) | `tui/src/views/AgentDetailView.tsx` | components | 9 useState, tabs, scroll, input, keyboard all in one |
| 12 | CostsView god component (613 lines) | `tui/src/views/CostsView.tsx` | components | Data fetching + sorting + 2 layout variants + detail view |
| 13 | 40+ hardcoded colors bypass theme | Multiple TUI files | styling | Views don't respect dark/light theme switching |
| 14 | Web: no code splitting | `web/src/App.tsx:1-16` | bundle | All 12 views eagerly imported |
| 15 | Web: no AbortController | `web/src/api/client.ts` | data-fetching | Memory leaks on rapid view switching |
| 16 | Web: usePolling race condition | `web/src/hooks/usePolling.ts:22` | data-fetching | Fetcher changes trigger constant polling |
| 17 | TUI ESLint version mismatch | `tui/package.json:31-38` | dx | typescript-eslint v7 mixed with @typescript-eslint v6 |
| 18 | TUI self-reference in deps | `tui/package.json:20` | bundle | `"@bc/tui": "."` circular dependency |
| 19 | Missing PageUp/PageDown navigation | `tui/src/hooks/useListNavigation.ts` | a11y | No fast navigation in long lists |
| 20 | Focus trap incomplete for overlays | `tui/navigation/FocusContext.tsx` | a11y | Keys leak through CommandBar/FilterBar |
| 21 | Light theme contrast insufficient | `tui/src/theme/themes.ts` | a11y | Gray on white < 4.5:1 ratio |
| 22 | useAgents debounce refs never cleaned | `tui/src/hooks/useAgents.ts:74` | state-mgmt | Memory leak as agents churn |
| 23 | Stale closure in useChannelsWithUnread | `tui/src/hooks/useChannels.ts:255` | state-mgmt | Expensive unread recalculation every render |
| 24 | 44 skipped tests (Jest mock migration) | Multiple TUI test files | testing | 15-20% of planned tests not executing |
| 25 | framer-motion adds ~35KB gzipped | `landing/package.json` | bundle | Heavy for a marketing site |
| 26 | Duplicate ThemeProvider implementations | `landing/src/app/_components/` vs `_contexts/` | components | Two theme systems, one broken |

## Minor Issues & Polish

| # | Issue | Location | Category | Impact |
|---|-------|----------|----------|--------|
| 27 | Duplicate ESLint config files | `tui/.eslintrc.cjs` + `tui/eslint.config.js` | dx | Conflicting lint rules |
| 28 | Namespace re-exports in barrel files | `tui/src/constants/index.ts` | bundle | Prevents tree-shaking |
| 29 | WebVitals only logs in development | `landing/src/app/_components/WebVitals.tsx` | performance | No production CWV monitoring |
| 30 | Web: hardcoded polling intervals | Multiple web views | data-fetching | Inconsistent refresh rates |
| 31 | Web: missing dynamic page titles | `web/index.html` | seo | All tabs show same title |
| 32 | Web: `text-bc-fg/80` undefined class | `web/src/views/Roles.tsx:65` | styling | Falls back to browser default |
| 33 | Web: no focus ring on nav links | `web/src/components/Layout.tsx` | a11y | Keyboard users can't see focus |
| 34 | Landing: redundant aria-labels | `landing/src/app/page.tsx` | a11y | Screen readers hear duplicate |
| 35 | Table.tsx array index as key | `tui/src/components/Table.tsx:66` | performance | Reconciliation issues on reorder |
| 36 | HOME env var used in regex unescaped | `tui/src/components/ActivityFeed.tsx:66` | security | Regex injection if HOME has special chars |
| 37 | RootProvider creates inline object | `tui/src/providers/RootProvider.tsx:35` | performance | ThemeProvider config recreated each render |
| 38 | FilterBar double state update | `tui/src/components/FilterBar.tsx:35` | performance | Two renders per keystroke |

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
- Design tokens via CSS custom properties
- WebSocket hook for real-time updates
- Consistent Tailwind usage with custom theme

**Landing:**
- Excellent SEO metadata (OG, Twitter, structured data)
- Proper `prefers-reduced-motion` handling for animations
- Responsive design with mobile-first breakpoints
- Playwright E2E tests configured
- Good aria-hidden usage on decorative icons
- Static export for CDN hosting

---

## Bundle Analysis

| App | Estimated Size | Issues |
|-----|---------------|--------|
| TUI | ~3,200 source lines + 3 deps | Self-reference dep, ESLint version mismatch |
| Web | ~2,400 source lines, no code splitting | All 12 views in initial bundle |
| Landing | ~4,000 source lines + framer-motion (~35KB gz) | Client-rendered page, heavy particle animation |

---

## Accessibility Assessment

**TUI (6/10):** Keyboard navigation works via j/k/arrows but missing PageUp/PageDown. Focus trapping incomplete for overlays. Light theme has poor contrast. No terminal size warning for small screens.

**Web (3/10):** No focus indicators on navigation. No skip-to-content. No ARIA landmarks. No keyboard shortcuts. Missing loading states for async operations.

**Landing (7/10):** Good aria-hidden on icons, proper aria-labels on CTAs, prefers-reduced-motion honored. Minor redundant labels. Form accessibility could improve.

---

## Performance Assessment

**TUI:** Context providers without memoized values cause cascade re-renders. FilterBar double-updates on each keystroke. Table uses array index keys. Agent debounce refs leak memory. Overall solid polling architecture.

**Web:** No code splitting means full bundle on first load. No AbortController causes memory leaks. usePolling can race-condition on fetcher change.

**Landing:** The biggest concern — entire page client-rendered with "use client". 80-particle O(n^2) animation on every frame. framer-motion adds ~35KB. Fonts loaded as render-blocking stylesheet.

---

## Action Plan

### Phase 1: Critical UX, Security & Bugs (immediate)
- Fix ThemeToggle `resolvedTheme` crash (landing)
- Fix API parameter injection (web)
- Guard `_setSpawnForTesting` export (TUI)
- Fix waitlist form error handling (landing)
- Add 404 route (web)
- Memoize ConfigContext and UnreadContext values (TUI)

### Phase 2: Performance (week 1)
- Convert landing page to Server Components
- Reduce AnimatedBackground particle count / optimize
- Add React.lazy code splitting to web routes
- Add AbortController to web API client
- Memoize RootProvider config object (TUI)
- Fix FilterBar double-update (TUI)

### Phase 3: Accessibility (week 2)
- Add PageUp/PageDown to TUI list navigation
- Fix focus trapping for TUI overlays
- Fix light theme contrast
- Add focus indicators to web navigation
- Add skip-to-content link (web)

### Phase 4: Component Architecture (week 3)
- Split AgentDetailView into sub-components
- Split CostsView into sub-components
- Resolve duplicate ThemeProvider (landing)
- Migrate hardcoded colors to theme system (TUI)

### Phase 5: State Management & Data Fetching (week 4)
- Fix useChannelsWithUnread stale closure
- Clean up useAgents debounce refs
- Fix usePolling race condition (web)
- Add proper error recovery/backoff to polling

### Phase 6: Testing & DX (week 5)
- Add tests for web dashboard
- Migrate TUI Jest mocks to bun:test
- Fix ESLint version mismatch
- Remove duplicate ESLint config
- Remove self-reference dep
- Add production Web Vitals reporting (landing)
