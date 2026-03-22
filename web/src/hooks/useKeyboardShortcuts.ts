import { useEffect, useCallback, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";

const NAV_PATHS = [
  "/",
  "/agents",
  "/channels",
  "/costs",
  "/roles",
  "/tools",
  "/mcp",
  "/cron",
  "/secrets",
] as const;

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName.toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return true;
  if ((el as HTMLElement).isContentEditable) return true;
  return false;
}

export function useKeyboardShortcuts() {
  const navigate = useNavigate();
  const location = useLocation();
  const [helpOpen, setHelpOpen] = useState(false);

  const toggleHelp = useCallback(() => {
    setHelpOpen((prev) => !prev);
  }, []);

  const closeHelp = useCallback(() => {
    setHelpOpen(false);
  }, []);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Always allow Escape to close help
      if (e.key === "Escape" && helpOpen) {
        e.preventDefault();
        closeHelp();
        return;
      }

      // Ignore when typing in inputs
      if (isInputFocused()) return;

      // Ignore when modifier keys are held (except Shift for ?)
      if (e.ctrlKey || e.metaKey || e.altKey) return;

      switch (e.key) {
        case "?": {
          e.preventDefault();
          toggleHelp();
          break;
        }

        case "/": {
          const searchInput = document.querySelector<HTMLInputElement>(
            'input[type="search"], input[placeholder*="earch"], input[data-search]',
          );
          if (searchInput) {
            e.preventDefault();
            searchInput.focus();
          }
          break;
        }

        case "j": {
          e.preventDefault();
          scrollListItem(1);
          break;
        }

        case "k": {
          e.preventDefault();
          scrollListItem(-1);
          break;
        }

        default: {
          // 1-9 for sidebar navigation
          const num = parseInt(e.key, 10);
          if (num >= 1 && num <= 9 && num <= NAV_PATHS.length) {
            const target = NAV_PATHS[num - 1];
            if (target && location.pathname !== target) {
              e.preventDefault();
              navigate(target);
            }
          }
          break;
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [navigate, location.pathname, helpOpen, toggleHelp, closeHelp]);

  return { helpOpen, closeHelp };
}

function scrollListItem(direction: 1 | -1) {
  // Find table rows or list items in the main content area
  const main = document.querySelector("main");
  if (!main) return;

  const rows = main.querySelectorAll<HTMLElement>(
    "tbody tr, [role='listitem'], [data-list-item]",
  );
  if (rows.length === 0) return;

  // Find currently highlighted row
  const highlighted = main.querySelector<HTMLElement>(
    "tr[data-kb-active], [data-kb-active]",
  );
  let currentIndex = -1;

  if (highlighted) {
    rows.forEach((row, i) => {
      if (row === highlighted) currentIndex = i;
    });
  }

  // Calculate next index
  let nextIndex: number;
  if (currentIndex === -1) {
    nextIndex = direction === 1 ? 0 : rows.length - 1;
  } else {
    nextIndex = currentIndex + direction;
    if (nextIndex < 0) nextIndex = 0;
    if (nextIndex >= rows.length) nextIndex = rows.length - 1;
  }

  // Remove previous highlight
  if (highlighted) {
    highlighted.removeAttribute("data-kb-active");
    highlighted.classList.remove("bg-bc-surface");
  }

  // Apply new highlight and scroll into view
  const nextRow = rows[nextIndex];
  if (nextRow) {
    nextRow.setAttribute("data-kb-active", "true");
    nextRow.classList.add("bg-bc-surface");
    nextRow.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }
}
