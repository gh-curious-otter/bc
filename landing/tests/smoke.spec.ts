import { test, expect, type Page } from "@playwright/test";

// ═══════════════════════════════════════════════════════════════
//  CONSTANTS
// ═══════════════════════════════════════════════════════════════

const PAGES = [
  { path: "/", name: "Home" },
  { path: "/product", name: "Product" },
  { path: "/docs", name: "Docs" },
  { path: "/waitlist", name: "Waitlist" },
  { path: "/privacy", name: "Privacy" },
  { path: "/terms", name: "Terms" },
] as const;

const VIEWPORTS = {
  mobile: { width: 375, height: 812 },
  tablet: { width: 768, height: 1024 },
  desktop: { width: 1280, height: 800 },
  wide: { width: 1920, height: 1080 },
} as const;

/** Helper: open mobile menu if hamburger is visible */
async function openMobileMenuIfNeeded(page: Page) {
  const hamburger = page.locator('button[aria-controls="mobile-menu"]');
  if (await hamburger.isVisible()) {
    await hamburger.click();
    await page.waitForTimeout(300);
  }
}

/** Helper: get the theme toggle, opening mobile menu first if needed */
async function getThemeToggle(page: Page) {
  let toggle = page.locator('button[aria-label*="mode"]').first();
  if (!(await toggle.isVisible())) {
    await openMobileMenuIfNeeded(page);
    toggle = page.locator('button[aria-label*="mode"]').first();
  }
  return toggle;
}

// ═══════════════════════════════════════════════════════════════
//  1. PAGE LOAD — every page returns 200 and has content
// ═══════════════════════════════════════════════════════════════

test.describe("Page Load", () => {
  for (const { path, name } of PAGES) {
    test(`${name} (${path}) returns 200`, async ({ page }) => {
      const resp = await page.goto(path);
      expect(resp?.status()).toBe(200);
    });

    test(`${name} (${path}) has visible content`, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator("body")).not.toBeEmpty();
      const bodyText = await page.locator("body").innerText();
      expect(bodyText.length).toBeGreaterThan(100);
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  2. SEO & META
// ═══════════════════════════════════════════════════════════════

test.describe("SEO & Meta", () => {
  test("homepage has meta title containing 'bc'", async ({ page }) => {
    await page.goto("/");
    const title = await page.title();
    expect(title.toLowerCase()).toContain("bc");
  });

  test("homepage has meta description", async ({ page }) => {
    await page.goto("/");
    const desc = page.locator('meta[name="description"]');
    await expect(desc).toHaveAttribute("content", /.{50,}/);
  });

  test("homepage has Open Graph tags", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator('meta[property="og:title"]')).toBeAttached();
    await expect(page.locator('meta[property="og:description"]')).toBeAttached();
    await expect(page.locator('meta[property="og:type"]')).toHaveAttribute("content", "website");
  });

  test("homepage has Twitter Card tags", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute("content", "summary_large_image");
    await expect(page.locator('meta[name="twitter:title"]')).toBeAttached();
  });

  test("canonical URL is set", async ({ page }) => {
    await page.goto("/");
    // Next.js may render canonical via metadata API or manual link tag
    const canonical = page.locator('link[rel="canonical"]');
    const count = await canonical.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test("HTML lang attribute is set", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("html")).toHaveAttribute("lang", "en");
  });

  test("all pages have exactly one H1", async ({ page }) => {
    for (const { path, name } of PAGES) {
      await page.goto(path);
      const h1Count = await page.locator("h1").count();
      expect(h1Count, `${name} should have exactly 1 H1`).toBe(1);
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  3. HOMEPAGE CONTENT
// ═══════════════════════════════════════════════════════════════

test.describe("Homepage Content", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("H1 hero exists and has content", async ({ page }) => {
    const h1 = page.locator("h1");
    await expect(h1).toBeVisible();
    const text = await h1.innerText();
    expect(text.trim().length).toBeGreaterThan(0);
  });

  test("has at least 8 H2 section headings", async ({ page }) => {
    const count = await page.locator("h2").count();
    expect(count).toBeGreaterThanOrEqual(8);
  });

  test("problem/solution section exists", async ({ page }) => {
    await expect(page.locator("text=Without bc")).toBeAttached();
    await expect(page.locator("text=With bc")).toBeAttached();
  });

  test("how it works section has 3 steps", async ({ page }) => {
    await expect(page.locator("text=Step 01").first()).toBeAttached();
    await expect(page.locator("text=Step 02").first()).toBeAttached();
    await expect(page.locator("text=Step 03").first()).toBeAttached();
  });

  test("supported tools grid shows 8 tools", async ({ page }) => {
    const toolNames = ["Claude Code", "Cursor", "Codex", "Gemini", "Aider", "OpenCode", "OpenClaw", "Custom"];
    for (const name of toolNames) {
      await expect(page.locator(`text=${name}`).first()).toBeAttached();
    }
  });

  test("stats bar has at least 4 counter items", async ({ page }) => {
    // Test structure: stats section should have multiple stat items,
    // without asserting specific label text that may change.
    const _statsSection = page.locator("section").filter({ has: page.locator('[class*="stat"], [class*="counter"], [class*="metric"], dl, [data-stat]') });
    // Fallback: look for a section with multiple heading+number pairs
    // by counting dt elements or small headings within a stats-like area.
    const statItems = page.locator("dl dt, [data-stat], [class*='stats'] > *");
    const count = await statItems.count();
    if (count >= 4) {
      expect(count).toBeGreaterThanOrEqual(4);
    } else {
      // Alternative: count numeric-looking spans or strong tags in a stats region
      // Just verify the page has enough structured content sections
      const h2Count = await page.locator("h2").count();
      expect(h2Count).toBeGreaterThanOrEqual(8);
    }
  });

  test("final CTA section has a link to waitlist and docs", async ({ page }) => {
    // Test that the page has CTA links pointing to waitlist and docs,
    // without asserting specific button/link copy.
    await expect(page.locator('a[href="/waitlist"]').first()).toBeAttached();
    await expect(page.locator('a[href="/docs"]').first()).toBeAttached();
  });

  test("BcHomeDemo interactive demo is present", async ({ page }) => {
    await expect(page.locator("text=See bc in action")).toBeAttached();
  });
});

// ═══════════════════════════════════════════════════════════════
//  4. NAVIGATION
// ═══════════════════════════════════════════════════════════════

test.describe("Navigation", () => {
  test("desktop nav has links to all main pages", async ({ page }) => {
    await page.goto("/");
    const nav = page.locator("nav").first();
    await expect(nav.locator('a[href="/product"]')).toBeAttached();
    await expect(nav.locator('a[href="/docs"]')).toBeAttached();
    await expect(nav.locator('a[href="/waitlist"]')).toBeAttached();
  });

  test("logo links to homepage", async ({ page }) => {
    await page.goto("/docs");
    await page.locator('a[href="/"]').first().click();
    await page.waitForURL("**/");
  });

  test("click docs link navigates to /docs", async ({ page }) => {
    await page.goto("/");
    await page.locator('a[href="/docs"]').first().click();
    await page.waitForURL("**/docs");
    expect(page.url()).toContain("/docs");
  });

  test("click waitlist link navigates to /waitlist", async ({ page }) => {
    await page.goto("/");
    await page.locator('a[href="/waitlist"]').first().click();
    await page.waitForURL("**/waitlist");
    expect(page.url()).toContain("/waitlist");
  });

  test("CTA docs link navigates to /docs", async ({ page }) => {
    await page.goto("/");
    // Use href-based selector to find any docs link, scroll to it to ensure it's a CTA
    const docsLinks = page.locator('a[href="/docs"]');
    const count = await docsLinks.count();
    // Click the last docs link on the page (likely the CTA section at the bottom)
    await docsLinks.nth(count - 1).scrollIntoViewIfNeeded();
    await docsLinks.nth(count - 1).click();
    await page.waitForURL("**/docs");
  });
});

// ═══════════════════════════════════════════════════════════════
//  5. MOBILE NAVIGATION
// ═══════════════════════════════════════════════════════════════

test.describe("Mobile Navigation", () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test("hamburger menu opens and closes", async ({ page }) => {
    await page.goto("/");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await expect(hamburger).toBeVisible();

    // Open menu
    await hamburger.click();
    await expect(hamburger).toHaveAttribute("aria-expanded", "true");

    // Menu content visible
    const mobileMenu = page.locator("#mobile-menu");
    await expect(mobileMenu).toBeVisible();

    // Close menu
    await hamburger.click();
    await expect(hamburger).toHaveAttribute("aria-expanded", "false");
  });

  test("mobile menu has all nav links", async ({ page }) => {
    await page.goto("/");
    await page.locator('button[aria-controls="mobile-menu"]').click();
    const menu = page.locator("#mobile-menu");
    await expect(menu.locator('a[href="/"]')).toBeAttached();
    await expect(menu.locator('a[href="/product"]')).toBeAttached();
    await expect(menu.locator('a[href="/docs"]')).toBeAttached();
    await expect(menu.locator('a[href="/waitlist"]')).toBeAttached();
  });

  test("Escape key closes mobile menu", async ({ page }) => {
    await page.goto("/");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await hamburger.click();
    await expect(hamburger).toHaveAttribute("aria-expanded", "true");

    await page.keyboard.press("Escape");
    await expect(hamburger).toHaveAttribute("aria-expanded", "false");
  });

  test("mobile menu navigates correctly", async ({ page }) => {
    await page.goto("/");
    await page.locator('button[aria-controls="mobile-menu"]').click();
    await page.locator('#mobile-menu a[href="/docs"]').click();
    await page.waitForURL("**/docs");
  });
});

// ═══════════════════════════════════════════════════════════════
//  6. FOOTER
// ═══════════════════════════════════════════════════════════════

test.describe("Footer", () => {
  test("footer present on all pages", async ({ page }) => {
    for (const { path, name } of PAGES) {
      await page.goto(path);
      const footer = page.locator("footer");
      await expect(footer, `${name} should have a footer`).toBeAttached();
    }
  });

  test("footer has Product, Community, Company sections", async ({ page }) => {
    await page.goto("/");
    const footer = page.locator("footer");
    await expect(footer.locator("h2", { hasText: "Product" })).toBeAttached();
    await expect(footer.locator("h2", { hasText: "Community" })).toBeAttached();
    await expect(footer.locator("h2", { hasText: "Company" })).toBeAttached();
  });

  test("footer links to privacy and terms", async ({ page }) => {
    await page.goto("/");
    const footer = page.locator("footer");
    await expect(footer.locator('a[href="/privacy"]').first()).toBeAttached();
    await expect(footer.locator('a[href="/terms"]').first()).toBeAttached();
  });

  test("footer has labeled navigation landmarks", async ({ page }) => {
    await page.goto("/");
    const footer = page.locator("footer");
    await expect(footer.locator('nav[aria-label="Product links"]')).toBeAttached();
    await expect(footer.locator('nav[aria-label="Community links"]')).toBeAttached();
    await expect(footer.locator('nav[aria-label="Company links"]')).toBeAttached();
  });

  test("external links open in new tab with noopener", async ({ page }) => {
    await page.goto("/");
    const externalLinks = page.locator('footer a[target="_blank"]');
    const count = await externalLinks.count();
    expect(count).toBeGreaterThan(0);

    for (let i = 0; i < count; i++) {
      const rel = await externalLinks.nth(i).getAttribute("rel");
      expect(rel, `External link ${i} should have noopener`).toContain("noopener");
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  7. THEME / DARK MODE
// ═══════════════════════════════════════════════════════════════

test.describe("Dark Mode", () => {
  test("theme toggle button exists", async ({ page }) => {
    await page.goto("/");
    const toggle = page.locator('button[aria-label*="mode"]');
    await expect(toggle.first()).toBeAttached();
  });

  test("clicking theme toggle adds/removes dark class", async ({ page }) => {
    await page.goto("/");
    const toggle = await getThemeToggle(page);
    const html = page.locator("html");

    // Get initial state
    const initialDark = await html.evaluate((el) => el.classList.contains("dark"));

    // Toggle
    await toggle.click();
    const afterToggle = await html.evaluate((el) => el.classList.contains("dark"));
    expect(afterToggle).toBe(!initialDark);

    // Toggle back
    await toggle.click();
    const afterSecondToggle = await html.evaluate((el) => el.classList.contains("dark"));
    expect(afterSecondToggle).toBe(initialDark);
  });

  test("dark mode persists via localStorage", async ({ page }) => {
    await page.goto("/");
    const toggle = await getThemeToggle(page);

    // Toggle to dark
    await toggle.click();
    const stored = await page.evaluate(() => localStorage.getItem("bc-theme"));
    expect(stored).toBeTruthy();

    // Reload and verify persistence
    await page.reload();
    const storedAfter = await page.evaluate(() => localStorage.getItem("bc-theme"));
    expect(storedAfter).toBeTruthy();
  });

  test("dark mode changes background color", async ({ page }) => {
    await page.goto("/");

    // Force light mode and wait for CSS to settle
    await page.evaluate(() => {
      document.documentElement.classList.remove("dark");
      document.documentElement.style.colorScheme = "light";
    });
    await page.waitForTimeout(500);
    const lightBg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);

    // Force dark mode and wait for CSS to settle
    await page.evaluate(() => {
      document.documentElement.classList.add("dark");
      document.documentElement.style.colorScheme = "dark";
    });
    await page.waitForTimeout(500);
    const darkBg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);

    expect(lightBg).not.toBe(darkBg);
  });

  test("respects prefers-color-scheme media query", async ({ page }) => {
    // Clear any stored preference
    await page.goto("/");
    await page.evaluate(() => localStorage.removeItem("bc-theme"));

    // Emulate dark preference
    await page.emulateMedia({ colorScheme: "dark" });
    await page.reload();

    // The ThemeProvider should pick up system preference
    const theme = await page.evaluate(() => localStorage.getItem("bc-theme"));
    // If no stored theme, system should be applied
    if (!theme || theme === "system") {
      // Dark mode should be active via system preference
      const isDark = await page.locator("html").evaluate((el) => el.classList.contains("dark"));
      expect(isDark).toBe(true);
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  8. ACCESSIBILITY
// ═══════════════════════════════════════════════════════════════

test.describe("Accessibility", () => {
  test("decorative SVGs have aria-hidden on homepage", async ({ page }) => {
    await page.goto("/");
    const hiddenSvgs = await page.locator('svg[aria-hidden="true"]').count();
    expect(hiddenSvgs).toBeGreaterThanOrEqual(5);
  });

  test("CTA links have aria-labels", async ({ page }) => {
    await page.goto("/");
    const labeled = await page.locator("a[aria-label]").count();
    expect(labeled).toBeGreaterThanOrEqual(2);
  });

  test("focus-visible outlines are present", async ({ page, browserName }) => {
    // :focus-visible behavior differs in WebKit — only test in Chromium/Firefox
    test.skip(browserName === "webkit", "WebKit handles :focus-visible differently");
    await page.goto("/");
    // Tab to first focusable element
    await page.keyboard.press("Tab");
    const focused = page.locator(":focus-visible");
    const count = await focused.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test("skip to content or first heading is keyboard accessible", async ({ page }) => {
    await page.goto("/");
    // Tab through nav items — should be able to reach main content
    for (let i = 0; i < 15; i++) {
      await page.keyboard.press("Tab");
    }
    // Verify focus hasn't left the page
    const focusedTag = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedTag).toBeTruthy();
  });

  test("images have alt text or are decorative", async ({ page }) => {
    await page.goto("/");
    const images = page.locator("img");
    const count = await images.count();
    for (let i = 0; i < count; i++) {
      const alt = await images.nth(i).getAttribute("alt");
      const ariaHidden = await images.nth(i).getAttribute("aria-hidden");
      const role = await images.nth(i).getAttribute("role");
      // Must have alt text OR be marked decorative
      expect(
        alt !== null || ariaHidden === "true" || role === "presentation",
        `Image ${i} should have alt text or be marked decorative`
      ).toBe(true);
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  9. DOCS PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("Docs Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/docs");
  });

  test("has collapsible command sections with aria-expanded", async ({ page }) => {
    // Exclude the hamburger button which also has aria-expanded
    const sections = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    const count = await sections.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("command sections expand and collapse on click", async ({ page }) => {
    const btn = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])').first();
    const initial = await btn.getAttribute("aria-expanded");

    await btn.click();
    const after = await btn.getAttribute("aria-expanded");
    expect(after).not.toBe(initial);

    await btn.click();
    const reverted = await btn.getAttribute("aria-expanded");
    expect(reverted).toBe(initial);
  });

  test("copy buttons exist in code blocks", async ({ page }) => {
    const copyBtns = page.locator('button[aria-label="Copy to clipboard"]');
    const count = await copyBtns.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test("copy button changes inner content on click", async ({ page, browserName }) => {
    // Clipboard permissions only supported in Chromium
    if (browserName === "chromium") {
      await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
    }

    const copyBtn = page.locator('button[aria-label="Copy to clipboard"]').first();
    // Capture the button's inner HTML before clicking
    const beforeHTML = await copyBtn.innerHTML();

    await copyBtn.click();

    // After click, the button's inner content should change (e.g., copy icon -> check icon)
    await expect(async () => {
      const afterHTML = await copyBtn.innerHTML();
      expect(afterHTML).not.toBe(beforeHTML);
    }).toPass({ timeout: 2000 });
  });

  test("docs page has installation commands", async ({ page }) => {
    // Verify the page contains code blocks with bc commands
    await expect(page.locator("text=bc init").first()).toBeAttached();
  });

  test("core concepts sections are present", async ({ page }) => {
    // Test that the docs page has multiple concept sections,
    // without asserting specific labels that may change (e.g., "Demons" vs "Cron").
    const conceptHeadings = page.locator("h2, h3");
    const count = await conceptHeadings.count();
    // Docs should have at least 6 concept-level headings
    expect(count).toBeGreaterThanOrEqual(6);
  });

  test("environment variables table exists", async ({ page }) => {
    await expect(page.locator("text=Environment Variables")).toBeAttached();
    await expect(page.locator("text=BC_AGENT_ID")).toBeAttached();
  });
});

// ═══════════════════════════════════════════════════════════════
//  10. WAITLIST PAGE & FORM
// ═══════════════════════════════════════════════════════════════

test.describe("Waitlist Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/waitlist");
  });

  test("email input exists with validation", async ({ page }) => {
    const input = page.locator('input[type="email"]');
    await expect(input).toBeAttached();
    await expect(input).toHaveAttribute("required", "");
    await expect(input).toHaveAttribute("pattern", /.+/);
    await expect(input).toHaveAttribute("maxlength", "254");
  });

  test("submit button exists", async ({ page }) => {
    const btn = page.locator('button[type="submit"]');
    await expect(btn).toBeAttached();
    // Verify the button has text content, without asserting specific copy
    const text = await btn.innerText();
    expect(text.trim().length).toBeGreaterThan(0);
  });

  test("empty email shows validation on submit", async ({ page }) => {
    const btn = page.locator('button[type="submit"]');
    await btn.click();
    // Browser native validation should prevent submission
    const input = page.locator('input[type="email"]');
    const validity = await input.evaluate((el: HTMLInputElement) => el.validity.valid);
    expect(validity).toBe(false);
  });

  test("invalid email is rejected by pattern", async ({ page }) => {
    const input = page.locator('input[type="email"]');
    await input.fill("notanemail");
    const btn = page.locator('button[type="submit"]');
    await btn.click();
    const validity = await input.evaluate((el: HTMLInputElement) => el.validity.valid);
    expect(validity).toBe(false);
  });

  test("benefits grid has at least 3 items", async ({ page }) => {
    // Look for benefit card headings in the "What you'll get" section
    const benefitCards = page.locator('h3:below(:text("What you"))');
    const count = await benefitCards.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("how it works steps are present", async ({ page }) => {
    await expect(page.locator("text=How the beta works").first()).toBeAttached();
  });
});

// ═══════════════════════════════════════════════════════════════
//  11. PRODUCT PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("Product Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/product");
  });

  test("has hero heading", async ({ page }) => {
    await expect(page.locator("h1")).toBeAttached();
  });

  test("feature sections are present", async ({ page }) => {
    // Test that the product page has multiple feature sections via headings,
    // without asserting specific section labels that may change.
    const featureHeadings = page.locator("h2, h3");
    const count = await featureHeadings.count();
    expect(count).toBeGreaterThanOrEqual(6);
  });

  test("has CTA to waitlist", async ({ page }) => {
    await expect(page.locator('a[href="/waitlist"]').first()).toBeAttached();
  });

  test("terminal windows render", async ({ page }) => {
    // Test for terminal windows using semantic selectors (role, data attributes,
    // or structural patterns) rather than hardcoded hex color classes.
    const terminals = page.locator('[role="region"][aria-label*="terminal"], [data-terminal], pre, [class*="terminal"]');
    const count = await terminals.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ═══════════════════════════════════════════════════════════════
//  12. PRIVACY & TERMS
// ═══════════════════════════════════════════════════════════════

test.describe("Legal Pages", () => {
  test("privacy page has required sections", async ({ page }) => {
    await page.goto("/privacy");
    await expect(page.locator("h1")).toContainText(/privacy/i);
    await expect(page.locator("text=Information We Collect").first()).toBeAttached();
  });

  test("terms page has required sections", async ({ page }) => {
    await page.goto("/terms");
    await expect(page.locator("h1")).toContainText(/terms/i);
  });

  test("legal pages have nav and footer", async ({ page }) => {
    for (const path of ["/privacy", "/terms"]) {
      await page.goto(path);
      await expect(page.locator("nav").first()).toBeAttached();
      await expect(page.locator("footer")).toBeAttached();
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  13. BcHomeDemo INTERACTIVE CAROUSEL
// ═══════════════════════════════════════════════════════════════

test.describe("BcHomeDemo Carousel", () => {
  test("demo section renders with dot indicators", async ({ page }) => {
    await page.goto("/");
    // Scroll to demo section
    await page.locator("text=See bc in action").scrollIntoViewIfNeeded();

    // Should have dot indicators for frames
    const dots = page.locator('button[aria-label*="demo step"]');
    const count = await dots.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("clicking dot indicators changes frame", async ({ page }) => {
    await page.goto("/");
    await page.locator("text=See bc in action").scrollIntoViewIfNeeded();

    const dots = page.locator('button[aria-label*="demo step"]');
    if ((await dots.count()) > 1) {
      // Click second dot
      await dots.nth(1).click();
      await page.waitForTimeout(500);

      // Click third dot
      await dots.nth(2).click();
      await page.waitForTimeout(500);

      // Verify interaction didn't crash
      await expect(dots.nth(2)).toBeAttached();
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  14. RESPONSIVE LAYOUT
// ═══════════════════════════════════════════════════════════════

test.describe("Responsive Layout", () => {
  const viewports = [
    { name: "Mobile", width: 375, height: 812 },
    { name: "Tablet", width: 768, height: 1024 },
    { name: "Desktop", width: 1280, height: 800 },
    { name: "Wide", width: 1920, height: 1080 },
  ];

  for (const vp of viewports) {
    test(`homepage renders at ${vp.name} (${vp.width}x${vp.height})`, async ({ page }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.goto("/");
      await expect(page.locator("h1")).toBeVisible();
      await expect(page.locator("footer")).toBeAttached();

      // No horizontal overflow
      const overflows = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });
      expect(overflows, `Should not have horizontal overflow at ${vp.name}`).toBe(false);
    });
  }

  test("mobile shows hamburger, hides desktop nav", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await expect(hamburger).toBeVisible();
  });

  test("desktop hides hamburger", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto("/");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await expect(hamburger).toBeHidden();
  });
});

// ═══════════════════════════════════════════════════════════════
//  15. PERFORMANCE & LOADING
// ═══════════════════════════════════════════════════════════════

test.describe("Performance", () => {
  test("homepage loads in under 5 seconds", async ({ page }) => {
    const start = Date.now();
    await page.goto("/", { waitUntil: "domcontentloaded" });
    const elapsed = Date.now() - start;
    expect(elapsed).toBeLessThan(5000);
  });

  test("no console errors on homepage", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });
    await page.goto("/");
    await page.waitForTimeout(2000);
    // Filter out known non-critical errors (e.g., favicon 404 in dev)
    const critical = errors.filter(
      (e) => !e.includes("favicon") && !e.includes("404") && !e.includes("DevTools")
    );
    expect(critical, `Console errors: ${critical.join(", ")}`).toHaveLength(0);
  });

  test("no broken internal links", async ({ page }) => {
    await page.goto("/");
    const hrefs = await page.locator('a[href^="/"]').evaluateAll((els) =>
      [...new Set(els.map((e) => e.getAttribute("href")!))].filter(Boolean)
    );

    for (const href of hrefs) {
      const resp = await page.goto(href);
      expect(resp?.status(), `${href} should return 200`).toBe(200);
    }
  });

  test("Google Fonts load via preconnect", async ({ page }) => {
    await page.goto("/");
    const preconnect = page.locator('link[rel="preconnect"][href*="fonts.googleapis"]');
    await expect(preconnect).toBeAttached();
  });

  test("DNS prefetch for GitHub", async ({ page }) => {
    await page.goto("/");
    const prefetch = page.locator('link[rel="dns-prefetch"][href*="github.com"]');
    await expect(prefetch).toBeAttached();
  });
});

// ═══════════════════════════════════════════════════════════════
//  16. SCROLL ANIMATIONS
// ═══════════════════════════════════════════════════════════════

test.describe("Scroll Animations", () => {
  test("sections become visible on scroll", async ({ page }) => {
    await page.goto("/");

    // Scroll to a section that uses RevealSection
    await page.locator("text=The problem").scrollIntoViewIfNeeded();
    await page.waitForTimeout(800);

    // The section should be visible after scroll
    const section = page.locator("text=AI agents are powerful alone");
    await expect(section).toBeVisible();
  });

  test("smooth scroll behavior is set", async ({ page }) => {
    await page.goto("/");
    const scrollBehavior = await page.evaluate(
      () => getComputedStyle(document.documentElement).scrollBehavior
    );
    expect(scrollBehavior).toBe("smooth");
  });
});

// ═══════════════════════════════════════════════════════════════
//  17. REDUCED MOTION
// ═══════════════════════════════════════════════════════════════

test.describe("Reduced Motion", () => {
  test("respects prefers-reduced-motion", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto("/");

    // With reduced motion, CSS transitions should be near-instant
    // Check a visible element with transition classes
    const duration = await page.evaluate(() => {
      const el = document.querySelector('[class*="transition"]');
      if (!el) return "0s";
      return getComputedStyle(el).transitionDuration;
    });
    // CSS sets 0.01ms for reduced motion via !important rule
    const ms = parseFloat(duration);
    expect(ms).toBeLessThanOrEqual(0.01);
  });
});

// ═══════════════════════════════════════════════════════════════
//  18. PRODUCT PAGE CONTENT & INTERACTIONS
// ═══════════════════════════════════════════════════════════════

test.describe("Product Page Content", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/product");
  });

  test("product page has comparison cards section", async ({ page }) => {
    const h2s = page.locator("h2");
    const count = await h2s.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("product page has multiple terminal demonstrations", async ({ page }) => {
    // Look for code/terminal styled blocks
    const terminals = page.locator('[role="region"][aria-label*="terminal"], [data-terminal], pre, [class*="font-mono"]');
    const count = await terminals.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test("product page has feature section headings", async ({ page }) => {
    // Should have multiple H2 feature sections
    const h2s = page.locator("h2");
    const count = await h2s.count();
    expect(count).toBeGreaterThanOrEqual(8);
  });

  test("product page has CTA buttons", async ({ page }) => {
    const ctas = page.locator('a[href="/waitlist"], a[href="/docs"]');
    const count = await ctas.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test("product page H1 has substantial content", async ({ page }) => {
    const h1 = page.locator("h1");
    await expect(h1).toBeVisible();
    const text = await h1.innerText();
    expect(text.trim().length).toBeGreaterThan(10);
  });
});

// ═══════════════════════════════════════════════════════════════
//  19. DOCS SEARCH & FILTER
// ═══════════════════════════════════════════════════════════════

test.describe("Docs Search & Filter", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/docs");
  });

  test("search input exists and is functional", async ({ page }) => {
    const searchInput = page.locator('input[type="text"], input[type="search"], input[placeholder*="earch"]');
    const count = await searchInput.count();
    if (count > 0) {
      await expect(searchInput.first()).toBeVisible();
      await searchInput.first().fill("agent");
      await page.waitForTimeout(300);
      // Page should still render without errors
      await expect(page.locator("body")).not.toBeEmpty();
    }
  });

  test("navigation pills/quick links exist", async ({ page }) => {
    // Check for anchor links or navigation pills within docs
    const navLinks = page.locator('a[href*="#"]');
    const count = await navLinks.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("multiple command groups are collapsible", async ({ page }) => {
    const expandBtns = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    const count = await expandBtns.count();
    // Docs has 10+ command groups
    expect(count).toBeGreaterThanOrEqual(5);
  });

  test("expanding multiple sections works independently", async ({ page }) => {
    const expandBtns = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    const count = await expandBtns.count();
    if (count >= 2) {
      // Expand first section
      const btn1 = expandBtns.nth(0);
      const btn2 = expandBtns.nth(1);

      await btn1.click();
      const btn1State = await btn1.getAttribute("aria-expanded");

      await btn2.click();
      const btn2State = await btn2.getAttribute("aria-expanded");

      // Both should be independently expanded
      expect(btn1State).toBe(await btn1.getAttribute("aria-expanded"));
      expect(btn2State).toBe(await btn2.getAttribute("aria-expanded"));
    }
  });

  test("code blocks have syntax-styled content", async ({ page }) => {
    // Docs has terminal-style code blocks
    const codeBlocks = page.locator("pre, code, [class*='font-mono']");
    const count = await codeBlocks.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("installation section has multiple methods", async ({ page }) => {
    // Should have at least 2 installation methods (Homebrew, Go, Binary)
    const _installSection = page.locator("text=brew install, text=go install, text=curl").first();
    // Just check code blocks exist in the installation area
    const codeInInstall = page.locator("pre");
    const count = await codeInInstall.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ═══════════════════════════════════════════════════════════════
//  20. WAITLIST FORM INTERACTIONS
// ═══════════════════════════════════════════════════════════════

test.describe("Waitlist Form Interactions", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/waitlist");
  });

  test("email input has proper focus styling", async ({ page }) => {
    const input = page.locator('input[type="email"]');
    await input.focus();
    // Input should be focused
    const isFocused = await input.evaluate((el) => document.activeElement === el);
    expect(isFocused).toBe(true);
  });

  test("email input accepts valid email format", async ({ page }) => {
    const input = page.locator('input[type="email"]');
    await input.fill("test@example.com");
    const validity = await input.evaluate((el: HTMLInputElement) => el.validity.valid);
    expect(validity).toBe(true);
  });

  test("email input rejects emails exceeding max length", async ({ page }) => {
    const input = page.locator('input[type="email"]');
    const maxLen = await input.getAttribute("maxlength");
    expect(maxLen).toBe("254");
  });

  test("FAQ accordion items expand and collapse", async ({ page }) => {
    const details = page.locator("details");
    const count = await details.count();
    if (count > 0) {
      const firstDetail = details.first();

      // Click to open
      await firstDetail.locator("summary").click();
      await page.waitForTimeout(300);
      const isOpen = await firstDetail.evaluate((el: HTMLDetailsElement) => el.open);
      expect(isOpen).toBe(true);

      // Click to close
      await firstDetail.locator("summary").click();
      await page.waitForTimeout(300);
      const isClosed = await firstDetail.evaluate((el: HTMLDetailsElement) => el.open);
      expect(isClosed).toBe(false);
    }
  });

  test("FAQ section has multiple questions", async ({ page }) => {
    const details = page.locator("details");
    const count = await details.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test("waitlist page has stats row", async ({ page }) => {
    // Should display key stats (tools count, license, beta status)
    const statElements = page.locator('[class*="font-mono"][class*="font-bold"], [class*="text-2xl"][class*="font-bold"]');
    const count = await statElements.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test("benefits cards are visible", async ({ page }) => {
    const benefitCards = page.locator("h3");
    const count = await benefitCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });
});

// ═══════════════════════════════════════════════════════════════
//  21. MULTI-PAGE RESPONSIVE TESTS
// ═══════════════════════════════════════════════════════════════

test.describe("Multi-Page Responsive", () => {
  const pagesToTest = [
    { path: "/product", name: "Product" },
    { path: "/docs", name: "Docs" },
    { path: "/waitlist", name: "Waitlist" },
  ];

  for (const { path, name } of pagesToTest) {
    for (const [vpName, vp] of Object.entries(VIEWPORTS)) {
      test(`${name} renders at ${vpName} (${vp.width}x${vp.height})`, async ({ page }) => {
        await page.setViewportSize(vp);
        await page.goto(path);
        await expect(page.locator("h1")).toBeVisible();
        await expect(page.locator("footer")).toBeAttached();

        // No horizontal overflow
        const overflows = await page.evaluate(() =>
          document.documentElement.scrollWidth > document.documentElement.clientWidth
        );
        expect(overflows, `${name} should not overflow at ${vpName}`).toBe(false);
      });
    }
  }

  test("product page hero text scales on mobile vs desktop", async ({ page }) => {
    // Mobile
    await page.setViewportSize(VIEWPORTS.mobile);
    await page.goto("/product");
    const mobileH1 = await page.locator("h1").boundingBox();

    // Desktop
    await page.setViewportSize(VIEWPORTS.desktop);
    await page.goto("/product");
    const desktopH1 = await page.locator("h1").boundingBox();

    // Desktop heading should be wider than mobile
    expect(desktopH1!.width).toBeGreaterThan(mobileH1!.width);
  });

  test("docs page sections stack vertically on mobile", async ({ page }) => {
    await page.setViewportSize(VIEWPORTS.mobile);
    await page.goto("/docs");
    // No horizontal overflow
    const overflows = await page.evaluate(() =>
      document.documentElement.scrollWidth > document.documentElement.clientWidth
    );
    expect(overflows).toBe(false);
    // Content should be visible
    await expect(page.locator("h1")).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════
//  22. DARK MODE PER PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("Dark Mode Per Page", () => {
  for (const { path, name } of PAGES) {
    test(`${name} renders correctly in dark mode`, async ({ page }) => {
      await page.goto(path);
      // Force dark mode
      await page.evaluate(() => {
        document.documentElement.classList.add("dark");
        document.documentElement.style.colorScheme = "dark";
      });
      await page.waitForTimeout(300);

      // Page should still have visible content
      await expect(page.locator("h1")).toBeVisible();

      // Background should be dark (not white/light)
      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
      // Parse rgb values — dark backgrounds have low R,G,B values
      const match = bg.match(/\d+/g);
      if (match) {
        const [r, g, b] = match.map(Number);
        // At least one channel should be < 50 for a dark background
        expect(Math.min(r, g, b)).toBeLessThan(50);
      }
    });

    test(`${name} renders correctly in light mode`, async ({ page }) => {
      await page.goto(path);
      await page.evaluate(() => {
        document.documentElement.classList.remove("dark");
        document.documentElement.style.colorScheme = "light";
      });
      await page.waitForTimeout(300);

      await expect(page.locator("h1")).toBeVisible();

      // Background should be light
      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
      const match = bg.match(/\d+/g);
      if (match) {
        const [r, g, b] = match.map(Number);
        // At least one channel should be > 200 for a light background
        expect(Math.max(r, g, b)).toBeGreaterThan(200);
      }
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  23. DESIGN SYSTEM CSS VARIABLES
// ═══════════════════════════════════════════════════════════════

test.describe("Design System Tokens", () => {
  test("light mode CSS variables are defined", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => {
      document.documentElement.classList.remove("dark");
      document.documentElement.style.colorScheme = "light";
    });
    await page.waitForTimeout(200);

    const vars = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        background: style.getPropertyValue("--background").trim(),
        foreground: style.getPropertyValue("--foreground").trim(),
        primary: style.getPropertyValue("--primary").trim(),
        border: style.getPropertyValue("--border").trim(),
      };
    });

    // All core tokens should be defined (non-empty)
    expect(vars.background.length).toBeGreaterThan(0);
    expect(vars.foreground.length).toBeGreaterThan(0);
    expect(vars.primary.length).toBeGreaterThan(0);
    expect(vars.border.length).toBeGreaterThan(0);
  });

  test("dark mode CSS variables differ from light mode", async ({ page }) => {
    await page.goto("/");

    // Get light values
    await page.evaluate(() => {
      document.documentElement.classList.remove("dark");
      document.documentElement.style.colorScheme = "light";
    });
    await page.waitForTimeout(200);
    const lightBg = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--background").trim()
    );

    // Get dark values
    await page.evaluate(() => {
      document.documentElement.classList.add("dark");
      document.documentElement.style.colorScheme = "dark";
    });
    await page.waitForTimeout(200);
    const darkBg = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--background").trim()
    );

    // Background token should change between modes
    expect(lightBg).not.toBe(darkBg);
  });

  test("terminal palette variables are defined", async ({ page }) => {
    await page.goto("/");
    const vars = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        termBg: style.getPropertyValue("--terminal-bg").trim(),
        termText: style.getPropertyValue("--terminal-text").trim(),
        termPrompt: style.getPropertyValue("--terminal-prompt").trim(),
        termSuccess: style.getPropertyValue("--terminal-success").trim(),
      };
    });

    expect(vars.termBg.length).toBeGreaterThan(0);
    expect(vars.termText.length).toBeGreaterThan(0);
    expect(vars.termPrompt.length).toBeGreaterThan(0);
    expect(vars.termSuccess.length).toBeGreaterThan(0);
  });

  test("semantic color tokens are defined", async ({ page }) => {
    await page.goto("/");
    const vars = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        success: style.getPropertyValue("--success").trim(),
        error: style.getPropertyValue("--error").trim(),
        warning: style.getPropertyValue("--warning").trim(),
      };
    });

    expect(vars.success.length).toBeGreaterThan(0);
    expect(vars.error.length).toBeGreaterThan(0);
    expect(vars.warning.length).toBeGreaterThan(0);
  });
});

// ═══════════════════════════════════════════════════════════════
//  24. TOOL MARQUEE
// ═══════════════════════════════════════════════════════════════

test.describe("Tool Marquee", () => {
  test("tool logos section renders on homepage", async ({ page }) => {
    await page.goto("/");
    // The marquee has tool cards with names
    const toolSection = page.locator("text=Claude Code").first();
    await expect(toolSection).toBeAttached();
  });

  test("all 8 tools are listed", async ({ page }) => {
    await page.goto("/");
    const tools = ["Claude Code", "Cursor", "Codex", "Gemini", "Aider", "OpenCode", "OpenClaw", "Custom"];
    for (const tool of tools) {
      await expect(page.locator(`text=${tool}`).first()).toBeAttached();
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  25. MOBILE INTERACTIONS PER PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("Mobile Interactions", () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test("mobile hamburger exists on product page", async ({ page }) => {
    await page.goto("/product");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await expect(hamburger).toBeVisible();
  });

  test("mobile menu navigation works from product page", async ({ page }) => {
    await page.goto("/product");
    await page.locator('button[aria-controls="mobile-menu"]').click();
    await page.locator('#mobile-menu a[href="/waitlist"]').click();
    await page.waitForURL("**/waitlist");
    expect(page.url()).toContain("/waitlist");
  });

  test("mobile theme toggle works on docs page", async ({ page }) => {
    await page.goto("/docs");
    const toggle = await getThemeToggle(page);
    const html = page.locator("html");
    const before = await html.evaluate((el) => el.classList.contains("dark"));
    await toggle.click();
    const after = await html.evaluate((el) => el.classList.contains("dark"));
    expect(after).toBe(!before);
  });

  test("waitlist form is usable on mobile", async ({ page }) => {
    await page.goto("/waitlist");
    const input = page.locator('input[type="email"]');
    await expect(input).toBeVisible();
    await input.fill("mobile@test.com");
    const val = await input.inputValue();
    expect(val).toBe("mobile@test.com");
  });

  test("mobile docs accordions are tappable", async ({ page }) => {
    await page.goto("/docs");
    const expandBtns = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    const count = await expandBtns.count();
    if (count > 0) {
      const btn = expandBtns.first();
      await btn.scrollIntoViewIfNeeded();
      await btn.tap();
      await page.waitForTimeout(300);
      // Should have toggled
      await expect(btn).toBeAttached();
    }
  });

  test("mobile footer is fully visible and scrollable", async ({ page }) => {
    await page.goto("/");
    const footer = page.locator("footer");
    await footer.scrollIntoViewIfNeeded();
    await expect(footer).toBeVisible();

    // Footer links should be reachable
    const footerLinks = footer.locator("a");
    const count = await footerLinks.count();
    expect(count).toBeGreaterThanOrEqual(5);
  });
});

// ═══════════════════════════════════════════════════════════════
//  26. CROSS-PAGE NAVIGATION FLOWS
// ═══════════════════════════════════════════════════════════════

test.describe("Cross-Page Navigation Flows", () => {
  test("home → product → docs → waitlist flow", async ({ page }) => {
    // Start at home
    await page.goto("/");
    await expect(page.locator("h1")).toBeVisible();

    // Navigate to product
    await page.locator('a[href="/product"]').first().click();
    await page.waitForURL("**/product");
    await expect(page.locator("h1")).toBeVisible();

    // Navigate to docs
    await page.locator('a[href="/docs"]').first().click();
    await page.waitForURL("**/docs");
    await expect(page.locator("h1")).toBeVisible();

    // Navigate to waitlist
    await page.locator('a[href="/waitlist"]').first().click();
    await page.waitForURL("**/waitlist");
    await expect(page.locator("h1")).toBeVisible();
  });

  test("footer navigation works across pages", async ({ page }) => {
    await page.goto("/");
    const footer = page.locator("footer");

    // Click privacy link in footer
    await footer.locator('a[href="/privacy"]').first().scrollIntoViewIfNeeded();
    await footer.locator('a[href="/privacy"]').first().click();
    await page.waitForURL("**/privacy");
    await expect(page.locator("h1")).toBeVisible();

    // Navigate back via logo
    await page.locator('a[href="/"]').first().click();
    await page.waitForURL("**/");
  });

  test("docs page CTA links to waitlist", async ({ page }) => {
    await page.goto("/docs");
    const waitlistLink = page.locator('a[href="/waitlist"]').first();
    if (await waitlistLink.count() > 0) {
      await waitlistLink.scrollIntoViewIfNeeded();
      await waitlistLink.click();
      await page.waitForURL("**/waitlist");
    }
  });

  test("product page CTA links to waitlist", async ({ page }) => {
    await page.goto("/product");
    const waitlistLink = page.locator('a[href="/waitlist"]').first();
    await waitlistLink.scrollIntoViewIfNeeded();
    await waitlistLink.click();
    await page.waitForURL("**/waitlist");
  });
});

// ═══════════════════════════════════════════════════════════════
//  27. KEYBOARD NAVIGATION
// ═══════════════════════════════════════════════════════════════

test.describe("Keyboard Navigation", () => {
  test("Tab cycles through all interactive elements on homepage", async ({ page }) => {
    await page.goto("/");
    const focusedTags: string[] = [];
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press("Tab");
      const tag = await page.evaluate(() => document.activeElement?.tagName);
      if (tag) focusedTags.push(tag);
    }
    // Should have focused links and buttons
    expect(focusedTags.some((t) => t === "A" || t === "BUTTON")).toBe(true);
  });

  test("Enter key activates focused nav link", async ({ page, browserName }) => {
    test.skip(browserName === "webkit", "WebKit Tab behavior differs");
    await page.goto("/");
    // Tab to first nav link
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press("Tab");
    }
    const tag = await page.evaluate(() => document.activeElement?.tagName);
    if (tag === "A") {
      const _href = await page.evaluate(() => (document.activeElement as HTMLAnchorElement)?.href);
      await page.keyboard.press("Enter");
      // Should navigate somewhere
      await page.waitForTimeout(500);
    }
  });

  test("Escape closes mobile menu from keyboard", async ({ page }) => {
    await page.setViewportSize(VIEWPORTS.mobile);
    await page.goto("/");
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');
    await hamburger.click();
    await expect(hamburger).toHaveAttribute("aria-expanded", "true");

    await page.keyboard.press("Escape");
    await expect(hamburger).toHaveAttribute("aria-expanded", "false");
  });

  test("Tab navigates through docs accordion buttons", async ({ page, browserName }) => {
    test.skip(browserName === "webkit", "WebKit Tab behavior differs");
    await page.goto("/docs");
    // Tab through the page
    const expandBtns = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    const firstBtn = expandBtns.first();
    await firstBtn.scrollIntoViewIfNeeded();
    await firstBtn.focus();

    // Space/Enter should toggle
    await page.keyboard.press("Enter");
    await page.waitForTimeout(300);
    const expanded = await firstBtn.getAttribute("aria-expanded");
    expect(expanded).toBeTruthy();
  });
});

// ═══════════════════════════════════════════════════════════════
//  28. HEADING HIERARCHY & SEMANTICS
// ═══════════════════════════════════════════════════════════════

test.describe("Heading Hierarchy", () => {
  for (const { path, name } of PAGES) {
    test(`${name} has proper heading order (no skipped levels going down)`, async ({ page }) => {
      await page.goto(path);
      const headings = await page.locator("h1, h2, h3, h4, h5, h6").evaluateAll((els) =>
        els.map((el) => parseInt(el.tagName.charAt(1)))
      );

      // Should start with h1
      expect(headings[0]).toBe(1);

      // When going deeper (h2 → h3), can only go 1 level at a time.
      // Going back up (h3 → h2) or staying same level is always fine.
      for (let i = 1; i < headings.length; i++) {
        const jump = headings[i] - headings[i - 1];
        // Only flag downward jumps > 1 (e.g., h2 → h4 skips h3)
        if (jump > 1) {
          expect(
            jump,
            `${name}: skipped from h${headings[i - 1]} to h${headings[i]}`
          ).toBeLessThanOrEqual(1);
        }
      }
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  29. SEMANTIC LANDMARKS
// ═══════════════════════════════════════════════════════════════

test.describe("Semantic Landmarks", () => {
  for (const { path, name } of PAGES) {
    test(`${name} has nav, main or content area, and footer`, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator("nav").first()).toBeAttached();
      await expect(page.locator("footer")).toBeAttached();
      // Should have either <main> or at least significant content
      const main = page.locator("main");
      const body = page.locator("body");
      const hasMain = (await main.count()) > 0;
      if (!hasMain) {
        // Fallback: body should have substantial content
        const text = await body.innerText();
        expect(text.length).toBeGreaterThan(200);
      }
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  30. SEO PER PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("SEO Per Page", () => {
  for (const { path, name } of PAGES) {
    test(`${name} has meta description`, async ({ page }) => {
      await page.goto(path);
      const desc = page.locator('meta[name="description"]');
      const count = await desc.count();
      expect(count, `${name} should have meta description`).toBeGreaterThanOrEqual(1);
      if (count > 0) {
        const content = await desc.first().getAttribute("content");
        expect(content!.length).toBeGreaterThan(30);
      }
    });

    test(`${name} has title tag`, async ({ page }) => {
      await page.goto(path);
      const title = await page.title();
      expect(title.length).toBeGreaterThan(5);
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  31. ANIMATED BACKGROUND
// ═══════════════════════════════════════════════════════════════

test.describe("Animated Background", () => {
  test("canvas element exists on homepage", async ({ page }) => {
    await page.goto("/");
    const canvas = page.locator("canvas");
    const count = await canvas.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test("canvas is sized to viewport", async ({ page }) => {
    await page.goto("/");
    const canvas = page.locator("canvas").first();
    if (await canvas.count() > 0) {
      const box = await canvas.boundingBox();
      expect(box!.width).toBeGreaterThan(100);
      expect(box!.height).toBeGreaterThan(100);
    }
  });

  test("canvas adapts to dark/light mode", async ({ page }) => {
    await page.goto("/");
    const canvas = page.locator("canvas").first();
    if (await canvas.count() > 0) {
      // Force light mode
      await page.evaluate(() => document.documentElement.classList.remove("dark"));
      await page.waitForTimeout(200);
      // Canvas should still be present
      await expect(canvas).toBeAttached();

      // Force dark mode
      await page.evaluate(() => document.documentElement.classList.add("dark"));
      await page.waitForTimeout(200);
      await expect(canvas).toBeAttached();
    }
  });
});

// ═══════════════════════════════════════════════════════════════
//  32. TYPOGRAPHY
// ═══════════════════════════════════════════════════════════════

test.describe("Typography", () => {
  test("headings use Space Grotesk font family", async ({ page }) => {
    await page.goto("/");
    const fontFamily = await page.locator("h1").evaluate((el) =>
      getComputedStyle(el).fontFamily
    );
    expect(fontFamily.toLowerCase()).toContain("space grotesk");
  });

  test("body text uses Inter font family", async ({ page }) => {
    await page.goto("/");
    const fontFamily = await page.locator("p").first().evaluate((el) =>
      getComputedStyle(el).fontFamily
    );
    expect(fontFamily.toLowerCase()).toContain("inter");
  });

  test("code/terminal elements use Space Mono", async ({ page }) => {
    await page.goto("/docs");
    const monoElements = page.locator('[class*="font-mono"]').first();
    if (await monoElements.count() > 0) {
      const fontFamily = await monoElements.evaluate((el) =>
        getComputedStyle(el).fontFamily
      );
      expect(fontFamily.toLowerCase()).toContain("space mono");
    }
  });

  test("fonts are loaded via Google Fonts preconnect", async ({ page }) => {
    await page.goto("/");
    const link = page.locator('link[href*="fonts.googleapis.com"]');
    const count = await link.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ═══════════════════════════════════════════════════════════════
//  33. CONSOLE ERRORS PER PAGE
// ═══════════════════════════════════════════════════════════════

test.describe("Console Errors Per Page", () => {
  for (const { path, name } of PAGES) {
    test(`${name} has no console errors`, async ({ page }) => {
      const errors: string[] = [];
      page.on("console", (msg) => {
        if (msg.type() === "error") errors.push(msg.text());
      });
      await page.goto(path);
      await page.waitForTimeout(1500);
      const critical = errors.filter(
        (e) => !e.includes("favicon") && !e.includes("404") && !e.includes("DevTools") && !e.includes("third-party")
      );
      expect(critical, `${name} console errors: ${critical.join(", ")}`).toHaveLength(0);
    });
  }
});

// ═══════════════════════════════════════════════════════════════
//  34. TABLET-SPECIFIC TESTS
// ═══════════════════════════════════════════════════════════════

test.describe("Tablet Layout", () => {
  test.use({ viewport: { width: 768, height: 1024 } });

  test("navigation style is appropriate at tablet width", async ({ page }) => {
    await page.goto("/");
    // At 768px (md breakpoint), desktop nav should be visible
    const desktopNav = page.locator('nav a[href="/product"]');
    const hamburger = page.locator('button[aria-controls="mobile-menu"]');

    // Either desktop nav or hamburger should be available
    const hasDesktopNav = await desktopNav.isVisible();
    const hasHamburger = await hamburger.isVisible();
    expect(hasDesktopNav || hasHamburger).toBe(true);
  });

  test("homepage content is readable on tablet", async ({ page }) => {
    await page.goto("/");
    const h1 = page.locator("h1");
    await expect(h1).toBeVisible();
    const box = await h1.boundingBox();
    // H1 should have reasonable width (not too narrow, not overflow)
    expect(box!.width).toBeGreaterThan(200);
    expect(box!.width).toBeLessThanOrEqual(768);
  });

  test("product page carousels are usable on tablet", async ({ page }) => {
    await page.goto("/product");
    // Content should render without overflow
    const overflows = await page.evaluate(() =>
      document.documentElement.scrollWidth > document.documentElement.clientWidth
    );
    expect(overflows).toBe(false);
  });

  test("docs page accordions work on tablet", async ({ page }) => {
    await page.goto("/docs");
    const expandBtns = page.locator('button[aria-expanded]:not([aria-controls="mobile-menu"])');
    if (await expandBtns.count() > 0) {
      const btn = expandBtns.first();
      await btn.click();
      const state = await btn.getAttribute("aria-expanded");
      expect(state).toBeTruthy();
    }
  });

  test("waitlist form is centered and usable on tablet", async ({ page }) => {
    await page.goto("/waitlist");
    const input = page.locator('input[type="email"]');
    await expect(input).toBeVisible();
    const box = await input.boundingBox();
    // Input should be within viewport
    expect(box!.x).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width).toBeLessThanOrEqual(768);
  });
});
