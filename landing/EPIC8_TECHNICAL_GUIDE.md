# Epic #8: Mobile Experience & Responsive Design - Technical Guide

## Overview
This guide provides eng-01 and eng-04 with implementation details, patterns, and review standards for making bc-landing fully responsive across mobile, tablet, and desktop.

**Target Breakpoints:**
- Mobile: 375px (iPhone SE)
- Tablet: 768px (iPad)
- Desktop: 1024px+

**Key Metrics:**
- Lighthouse score: >90
- LCP on mobile: <2.5s
- FCP on mobile: <2s
- Touch target minimum: 44x44px
- Min font size: 14px (WCAG AA)

---

## Task Assignments & Scope

### Task #29: Make Product Demos Responsive & Mobile-Optimized
**File:** `src/app/_components/ProductDemos.tsx`
**Complexity:** Medium-Large
**Estimated Scope:** 1-2 days
**Lead:** eng-01

**Current Issues:**
- Grid layout: `lg:grid-cols-[400px_1fr]` needs mobile fallback
- Terminal display: `min-h-[440px]` fixed height issues on mobile
- Step indicators: May not meet 44x44px minimum touch target
- Text sizes: Hardcoded values in terminal content
- Sticky positioning: `lg:sticky lg:top-24` works only on large screens

**Implementation Tasks:**
1. Add mobile breakpoint for 2-column → 1-column stacking
2. Replace fixed heights with responsive min-height values
3. Scale terminal display:
   - Mobile: min-h-[280px], text-[11px]
   - Tablet: min-h-[360px], text-[12px]
   - Desktop: min-h-[440px], text-[13px]
4. Update step indicators to meet 44x44px minimum
5. Add touch event handling to disable auto-advance on user interaction
6. Test animations on mobile (no jank on touch)

**Code Pattern Example:**
```tsx
// Mobile-first approach
<div className="grid gap-16 grid-cols-1 md:grid-cols-2 lg:grid-cols-[400px_1fr]">
  <div className="lg:sticky lg:top-24">
    {/* Narrative box */}
  </div>
  <div className="min-h-[280px] md:min-h-[360px] lg:min-h-[440px]">
    {/* Terminal display */}
  </div>
</div>
```

**Acceptance Criteria Checklist:**
- [ ] Renders correctly at 375px, 768px, 1024px+
- [ ] Terminal display readable on mobile
- [ ] Step indicators: h-12 w-12 minimum (48px for better touch)
- [ ] No text overflow on mobile
- [ ] Animations smooth on mobile (60fps target)
- [ ] Touch auto-advance doesn't jank
- [ ] All text >= 14px except decorative mono
- [ ] Lighthouse Performance >90
- [ ] Screenshots provided for all breakpoints

---

### Task #21: Refactor Mobile Navigation Menu
**File:** `src/app/_components/Nav.tsx`
**Complexity:** Low-Medium
**Estimated Scope:** 4-6 hours
**Lead:** eng-04

**Current Issues:**
- Missing links: Need to add Home, Waitlist, GitHub, Privacy, Terms
- Menu structure needs expansion
- Test on actual devices (iOS/Android)

**Navigation Structure (Target):**
```
Mobile Menu:
├── Product
├── Docs
├── Waitlist
├── GitHub
├── Contact
└── Separators: Privacy / Terms (footer style)
```

**Implementation Tasks:**
1. Update links array to include: Waitlist, GitHub, and external links
2. Verify hamburger button is h-11 w-11 (44x44px) ✓ Already correct
3. Ensure menu items have py-3 (at least 44px height including padding)
4. Test menu closes on link click ✓ Already working
5. Add keyboard navigation (Escape to close)
6. Test z-index layering (z-50 should be sufficient)
7. Test on actual iOS and Android devices

**Code Review Checklist:**
- [ ] h-11 w-11 button verified (44x44px)
- [ ] Menu items: py-3 minimum (ensures 44px height)
- [ ] All required links present
- [ ] ARIA labels: aria-label, aria-expanded, aria-controls
- [ ] Keyboard: Escape closes menu
- [ ] Z-index: z-50 is appropriate
- [ ] Focus visible states present
- [ ] No console errors/warnings
- [ ] Tested on iOS (Safari) and Android (Chrome)
- [ ] Animation: smooth (0.2s duration)
- [ ] Mobile screenshots provided

---

### Task #34: Mobile Typography & Readability
**File:** All components, focus on ProductDemos and ProductCarouselDemos
**Complexity:** Low-Medium
**Estimated Scope:** 4-6 hours
**Lead:** eng-01

**Current Issues:**
- Terminal fonts hardcoded: text-[10px], text-[12px], text-[13px]
- Some text < 14px minimum (WCAG AA)
- Heading sizes may not scale properly on mobile

**Implementation Strategy:**
1. Audit all text sizes in:
   - ProductDemos.tsx
   - ProductCarouselDemos.tsx
   - Forms (waitlist)
2. Update to responsive classes:
   - Body text: text-sm md:text-base (14px → 16px)
   - Terminal: text-[11px] md:text-[12px] lg:text-[13px]
   - Labels: text-xs md:text-sm
3. Verify WCAG AA contrast ratios
4. Test on actual small devices

**Tailwind Responsive Pattern:**
```tsx
// Mobile-first: smallest first
<span className="text-[11px] md:text-[12px] lg:text-[13px]">Terminal</span>
<p className="text-sm md:text-base leading-relaxed">Body text</p>
<h2 className="text-3xl md:text-4xl lg:text-5xl">Heading</h2>
```

**Acceptance Criteria Checklist:**
- [ ] All body text >= 14px
- [ ] Terminal mono fonts readable on mobile
- [ ] Headings scale: h1: 32px-56px, h3: 24px-32px
- [ ] Line heights >= 1.5
- [ ] No horizontal scrolling from text overflow
- [ ] Lighthouse readability > 90
- [ ] Tested on iPhone SE, iPad, Desktop
- [ ] WCAG AA contrast verified
- [ ] Screenshots at all breakpoints

---

### Task #35: Responsive Layout Audit
**File:** All layout components
**Complexity:** Low-Medium
**Estimated Scope:** 6-8 hours
**Lead:** eng-04

**Components to Audit:**
1. ProductDemos: `lg:grid-cols-[400px_1fr]`
2. ProductCarouselDemos: `lg:grid-cols-[320px_1fr]` (line 157, 206)
3. Waitlist form: padding, grid layout
4. BcHomeDemo: any fixed layouts
5. All max-w-* containers

**Testing Checklist:**
```
Breakpoints to test:
✓ 375px (iPhone SE) - mobile
✓ 768px (iPad) - tablet
✓ 1024px+ (Desktop)
✓ Resize browser window - check for layout shift
```

**Common Issues to Fix:**
- Fixed widths preventing responsiveness
- Columns not stacking on mobile
- Padding too large for small screens
- Grid gaps creating overflow

**Pattern to Verify:**
```tsx
// Good: mobile-first responsive
<div className="max-w-7xl mx-auto px-6">
  <div className="grid gap-16 grid-cols-1 md:grid-cols-2 lg:grid-cols-[400px_1fr]">
```

**Acceptance Criteria Checklist:**
- [ ] All pages render at 375px, 768px, 1024px
- [ ] No layout shift on browser resize
- [ ] No horizontal scrolling
- [ ] Fixed heights/widths removed where possible
- [ ] Padding appropriate for mobile (px-6 usually works)
- [ ] Container widths responsive (max-w-*)
- [ ] Grid/flex: grid-cols-1 → md:grid-cols-2 pattern
- [ ] Screenshots showing all breakpoints
- [ ] DevTools responsive mode + real device testing

---

### Task #36: Touch-Friendly Interactions
**File:** All interactive components
**Complexity:** Low
**Estimated Scope:** 4-6 hours
**Lead:** eng-01

**Components to Audit:**
1. ProductDemos step indicators (line 134-147)
2. ProductCarouselDemos controls (Prev/Next/Play buttons, line 391-408)
3. Carousel dots (line 435-445)
4. Nav menu items
5. Form buttons
6. All clickable links

**Touch Target Checklist:**
- [ ] Minimum 44x44px tap target (WCAG 2.1 Level AAA)
- [ ] 8px spacing between targets minimum
- [ ] No accidental adjacent taps

**Current Component Analysis:**
```tsx
// Step indicator example (line 135)
<button className="group relative flex h-1 flex-1 items-center">
// ❌ Problem: h-1 is too small (4px)
// ✓ Solution: h-12 w-12 (48px) on mobile

// Fix:
<button className="group relative flex h-12 w-12 md:h-1 md:flex-1 items-center">
```

**Implementation Pattern:**
```tsx
// Mobile: explicit button size
// Desktop: visual progress bar
<button className="h-12 w-12 md:h-1 md:flex-1 md:items-center">
  <div className="h-full w-full rounded-full transition-all" />
</button>
```

**Acceptance Criteria Checklist:**
- [ ] All buttons/links: 44x44px minimum
- [ ] Step indicators: 48x48px (12 + padding)
- [ ] 8px spacing between targets
- [ ] Focus states visible (focus-visible:ring)
- [ ] Keyboard navigation working (Tab, Enter, Space)
- [ ] Tested on iOS Safari and Android Chrome
- [ ] Touch doesn't trigger hover states incorrectly
- [ ] No jank on touch/scroll interactions
- [ ] Screenshots from real device testing

---

## Code Review Checklist

### Performance
- [ ] LCP <2.5s on mobile
- [ ] FCP <2s on mobile
- [ ] Lighthouse Performance >90
- [ ] No layout shifts (CLS <0.1)
- [ ] Images optimized for mobile

### Responsiveness
- [ ] Tested at 375px, 768px, 1024px
- [ ] All breakpoints use Tailwind's md:, lg: prefixes
- [ ] Mobile-first approach (smallest to largest)
- [ ] No horizontal scrolling
- [ ] Content readable on all sizes

### Accessibility
- [ ] Touch targets >= 44x44px
- [ ] Text >= 14px (WCAG AA)
- [ ] Contrast ratios meet WCAG AA
- [ ] ARIA labels present where needed
- [ ] Keyboard navigation works
- [ ] Focus visible states visible

### Code Quality
- [ ] No hardcoded pixel sizes (use Tailwind)
- [ ] Consistent spacing (scale of 4px: px-4, py-8, etc.)
- [ ] Responsive classes used consistently
- [ ] No inline styles
- [ ] Comments for complex responsive logic

### Testing
- [ ] Tested on actual mobile devices (iOS + Android)
- [ ] Browser DevTools responsive mode verified
- [ ] All breakpoints screenshot evidence
- [ ] Accessibility audited (aXe, WAVE)
- [ ] Performance tested (Lighthouse)

---

## Deployment Checklist

Before merging to main:
1. [ ] All acceptance criteria met
2. [ ] Code review approved
3. [ ] Performance benchmarks met
4. [ ] Mobile screenshots provided
5. [ ] No console errors or warnings
6. [ ] Tested on real devices
7. [ ] PR linked to GitHub issue
8. [ ] Updated CHANGELOG if applicable

---

## Testing Strategy

### Manual Testing
```
Devices to test:
- iPhone SE (375px)
- iPad (768px)
- Macbook/Desktop (1440px+)
```

### Browser Testing
```
Desktop:
- Chrome (latest)
- Firefox (latest)
- Safari (latest)

Mobile:
- iOS Safari
- Android Chrome
```

### Performance Testing
```
Use Lighthouse CI:
1. Run: lighthouse --view
2. Target scores: >90
3. Metrics: LCP <2.5s, FCP <2s, CLS <0.1
```

### Accessibility Testing
```
Tools:
- aXe DevTools browser extension
- WAVE (WebAIM)
- Manual keyboard navigation
- Manual screen reader testing (if possible)
```

---

## Git Workflow

```bash
# Create feature branch
git checkout -b task/29-product-demos-responsive

# Commit frequently with clear messages
git add .
git commit -m "feat: add mobile stacking for ProductDemos grid"

# Before PR: ensure clean history
git log --oneline  # Review commits

# Push to remote
git push origin task/29-product-demos-responsive

# Create PR with template:
# - Links to task (#29)
# - Describes changes
# - Links screenshots for all breakpoints
```

---

## Questions & Support

- **Blockers:** Post in #engineering thread
- **Code review questions:** Tag @tech-lead-02
- **Design questions:** Ask in thread
- **Performance issues:** Profile with DevTools, discuss in #engineering

Ready to start? Pull your assigned tasks from the queue and get going! 🚀
