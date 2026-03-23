"use client";

import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import { useState, useMemo, useEffect, useCallback, useRef } from "react";
import type { CommandGroup, CliCommand } from "@/lib/cli-docs";
import type { DocsSection } from "@/lib/docs-loader";
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  Search,
  BookOpen,
  Compass,
  FileText,
  Lightbulb,
  Terminal,
  Menu,
  X,
  Construction,
} from "lucide-react";

/* ═══════════════════════════════════════════════════════════════════
   TYPES
   ═══════════════════════════════════════════════════════════════════ */

interface NavItem {
  id: string;
  label: string;
  type: "article" | "cli-group" | "cli-command" | "placeholder";
  sectionId: string;
  content?: string;
  description?: string;
  cliGroup?: CommandGroup;
  cliCommand?: CliCommand;
}

interface NavSection {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  items: NavItem[];
}

interface TocHeading {
  id: string;
  text: string;
  level: number;
}

/* ═══════════════════════════════════════════════════════════════════
   COPY BUTTON
   ═══════════════════════════════════════════════════════════════════ */

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => {
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }}
      className="absolute right-3 top-3 rounded-md border border-white/10 bg-white/5 p-1.5 text-terminal-muted hover:text-terminal-text transition-colors"
      aria-label="Copy to clipboard"
    >
      {copied ? (
        <Check
          className="h-3.5 w-3.5 text-terminal-success"
          aria-hidden="true"
        />
      ) : (
        <Copy className="h-3.5 w-3.5" aria-hidden="true" />
      )}
    </button>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   CODE BLOCK
   ═══════════════════════════════════════════════════════════════════ */

function CodeBlock({
  code,
  language,
  title,
}: {
  code: string;
  language?: string;
  title?: string;
}) {
  return (
    <div className="group relative overflow-hidden rounded-lg border border-border bg-[#0C0A08] my-4">
      {title && (
        <div className="border-b border-white/[0.06] bg-[#151210] px-4 py-2 text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted flex items-center justify-between">
          <span>{title}</span>
          {language && (
            <span className="text-terminal-muted/50">{language}</span>
          )}
        </div>
      )}
      <div className="relative p-4 font-mono text-[13px] leading-relaxed text-terminal-text overflow-x-auto">
        <CopyButton text={code} />
        <pre>
          <code>{code}</code>
        </pre>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   INLINE MARKDOWN
   ═══════════════════════════════════════════════════════════════════ */

interface InlineMatch {
  type: string;
  pre: string;
  content: string;
  post: string;
  href?: string;
  idx: number;
}

function InlineMarkdown({ text }: { text: string }) {
  const parts: React.ReactNode[] = [];
  let remaining = text;
  let partKey = 0;

  while (remaining.length > 0) {
    const codeMatch = remaining.match(/^(.*?)`([^`]+)`(.*)$/);
    const boldMatch = remaining.match(/^(.*?)\*\*([^*]+)\*\*(.*)$/);
    const italicMatch = remaining.match(/^(.*?)(?<!\*)\*([^*]+)\*(?!\*)(.*)$/);
    const linkMatch = remaining.match(/^(.*?)\[([^\]]+)\]\(([^)]+)\)(.*)$/);

    const candidates: InlineMatch[] = [];

    if (codeMatch) {
      candidates.push({
        type: "code",
        pre: codeMatch[1],
        content: codeMatch[2],
        post: codeMatch[3],
        idx: codeMatch[1].length,
      });
    }
    if (boldMatch) {
      candidates.push({
        type: "bold",
        pre: boldMatch[1],
        content: boldMatch[2],
        post: boldMatch[3],
        idx: boldMatch[1].length,
      });
    }
    if (italicMatch) {
      candidates.push({
        type: "italic",
        pre: italicMatch[1],
        content: italicMatch[2],
        post: italicMatch[3],
        idx: italicMatch[1].length,
      });
    }
    if (linkMatch) {
      candidates.push({
        type: "link",
        pre: linkMatch[1],
        content: linkMatch[2],
        post: linkMatch[4],
        href: linkMatch[3],
        idx: linkMatch[1].length,
      });
    }

    // Pick the candidate with the earliest position
    candidates.sort((a, b) => a.idx - b.idx);
    const earliest = candidates.length > 0 ? candidates[0] : null;

    if (!earliest) {
      parts.push(<span key={partKey++}>{remaining}</span>);
      break;
    }

    if (earliest.pre) {
      parts.push(<span key={partKey++}>{earliest.pre}</span>);
    }

    if (earliest.type === "code") {
      parts.push(
        <code
          key={partKey++}
          className="rounded bg-muted px-1.5 py-0.5 text-[0.85em] font-mono text-primary"
        >
          {earliest.content}
        </code>,
      );
    } else if (earliest.type === "bold") {
      parts.push(
        <strong key={partKey++} className="font-semibold text-foreground">
          {earliest.content}
        </strong>,
      );
    } else if (earliest.type === "italic") {
      parts.push(
        <em key={partKey++} className="italic">
          {earliest.content}
        </em>,
      );
    } else if (earliest.type === "link") {
      parts.push(
        <a
          key={partKey++}
          href={earliest.href}
          className="text-primary hover:underline"
          target={earliest.href?.startsWith("http") ? "_blank" : undefined}
          rel={
            earliest.href?.startsWith("http")
              ? "noopener noreferrer"
              : undefined
          }
        >
          {earliest.content}
        </a>,
      );
    }

    remaining = earliest.post;
  }

  return <>{parts}</>;
}

/* ═══════════════════════════════════════════════════════════════════
   MARKDOWN RENDERER
   ═══════════════════════════════════════════════════════════════════ */

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function MarkdownContent({
  content,
  onHeadings,
}: {
  content: string;
  onHeadings?: (headings: TocHeading[]) => void;
}) {
  const lines = content.split("\n");
  const elements: React.ReactNode[] = [];
  const headings: TocHeading[] = [];
  let i = 0;
  let key = 0;
  let skipFirstH1 = true;

  while (i < lines.length) {
    const line = lines[i];

    // Skip the first H1
    if (line.startsWith("# ") && skipFirstH1) {
      skipFirstH1 = false;
      i++;
      if (i < lines.length && lines[i].trim() === "") i++;
      // Skip description line
      if (
        i < lines.length &&
        lines[i].trim() &&
        !lines[i].startsWith("#") &&
        !lines[i].startsWith("```")
      ) {
        i++;
      }
      continue;
    }

    // Code blocks
    if (line.startsWith("```")) {
      const langMatch = line.match(/^```(\w+)?/);
      const language = langMatch?.[1] || "";
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++;
      const code = codeLines.join("\n");
      if (code.trim()) {
        elements.push(
          <CodeBlock key={key++} code={code} language={language} />,
        );
      }
      continue;
    }

    // Tables
    if (line.includes("|") && line.trim().startsWith("|")) {
      const tableRows: string[] = [];
      while (
        i < lines.length &&
        lines[i].includes("|") &&
        lines[i].trim().startsWith("|")
      ) {
        tableRows.push(lines[i]);
        i++;
      }
      if (tableRows.length >= 2) {
        const parseRow = (row: string) =>
          row
            .split("|")
            .slice(1, -1)
            .map((c) => c.trim());
        const headerCells = parseRow(tableRows[0]);
        // Skip separator row (row[1])
        const bodyRows = tableRows.slice(2).map(parseRow);
        elements.push(
          <div key={key++} className="my-4 overflow-x-auto">
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="border-b border-border">
                  {headerCells.map((cell, ci) => (
                    <th
                      key={ci}
                      className="text-left px-3 py-2 font-semibold text-foreground text-xs uppercase tracking-wider"
                    >
                      {cell}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {bodyRows.map((row, ri) => (
                  <tr key={ri} className="border-b border-border/50">
                    {row.map((cell, ci) => (
                      <td
                        key={ci}
                        className="px-3 py-2 text-muted-foreground"
                      >
                        <InlineMarkdown text={cell} />
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>,
        );
      }
      continue;
    }

    // H2 headings
    if (line.startsWith("## ")) {
      const text = line.replace("## ", "");
      const id = slugify(text);
      headings.push({ id, text, level: 2 });
      elements.push(
        <h2
          key={key++}
          id={id}
          className="text-xl font-bold tracking-tight mt-10 mb-4 text-foreground scroll-mt-20"
        >
          {text}
        </h2>,
      );
      i++;
      continue;
    }

    // H3 headings
    if (line.startsWith("### ")) {
      const text = line.replace("### ", "");
      const id = slugify(text);
      headings.push({ id, text, level: 3 });
      elements.push(
        <h3
          key={key++}
          id={id}
          className="text-base font-bold tracking-tight mt-8 mb-3 text-foreground scroll-mt-20"
        >
          {text}
        </h3>,
      );
      i++;
      continue;
    }

    // H4 headings
    if (line.startsWith("#### ")) {
      const text = line.replace("#### ", "");
      const id = slugify(text);
      headings.push({ id, text, level: 4 });
      elements.push(
        <h4
          key={key++}
          id={id}
          className="text-sm font-bold tracking-tight mt-6 mb-2 text-foreground scroll-mt-20"
        >
          {text}
        </h4>,
      );
      i++;
      continue;
    }

    // Blockquotes (rendered as callout boxes)
    if (line.startsWith("> ")) {
      const quoteLines: string[] = [];
      while (i < lines.length && lines[i].startsWith("> ")) {
        quoteLines.push(lines[i].replace(/^>\s*/, ""));
        i++;
      }
      const quoteText = quoteLines.join(" ");
      // Detect admonition type
      const isWarning =
        quoteText.toLowerCase().startsWith("warning") ||
        quoteText.toLowerCase().startsWith("caution");
      const isTip =
        quoteText.toLowerCase().startsWith("tip") ||
        quoteText.toLowerCase().startsWith("note");
      const borderColor = isWarning
        ? "border-warning"
        : isTip
          ? "border-info"
          : "border-primary/30";
      const bgColor = isWarning
        ? "bg-warning/5"
        : isTip
          ? "bg-info/5"
          : "bg-primary/5";
      elements.push(
        <div
          key={key++}
          className={`border-l-4 ${borderColor} ${bgColor} rounded-r-lg pl-4 pr-4 py-3 my-4`}
        >
          <p className="text-sm text-muted-foreground leading-relaxed">
            <InlineMarkdown text={quoteText} />
          </p>
        </div>,
      );
      continue;
    }

    // Unordered list items
    if (line.match(/^[-*]\s/)) {
      const listItems: string[] = [];
      while (i < lines.length && lines[i].match(/^[-*]\s/)) {
        listItems.push(lines[i].replace(/^[-*]\s/, ""));
        i++;
      }
      elements.push(
        <ul key={key++} className="list-disc pl-5 my-3 space-y-1.5">
          {listItems.map((item, idx) => (
            <li
              key={idx}
              className="text-sm text-muted-foreground leading-relaxed"
            >
              <InlineMarkdown text={item} />
            </li>
          ))}
        </ul>,
      );
      continue;
    }

    // Ordered list items
    if (line.match(/^\d+\.\s/)) {
      const listItems: string[] = [];
      while (i < lines.length && lines[i].match(/^\d+\.\s/)) {
        listItems.push(lines[i].replace(/^\d+\.\s/, ""));
        i++;
      }
      elements.push(
        <ol key={key++} className="list-decimal pl-5 my-3 space-y-1.5">
          {listItems.map((item, idx) => (
            <li
              key={idx}
              className="text-sm text-muted-foreground leading-relaxed"
            >
              <InlineMarkdown text={item} />
            </li>
          ))}
        </ol>,
      );
      continue;
    }

    // Horizontal rule
    if (line.match(/^---+$/)) {
      elements.push(
        <hr key={key++} className="border-border my-8" />,
      );
      i++;
      continue;
    }

    // Blank lines
    if (!line.trim()) {
      i++;
      continue;
    }

    // Regular paragraph
    elements.push(
      <p
        key={key++}
        className="text-sm text-muted-foreground leading-relaxed my-2"
      >
        <InlineMarkdown text={line} />
      </p>,
    );
    i++;
  }

  // Report headings for TOC
  useEffect(() => {
    if (onHeadings) {
      onHeadings(headings);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [content]);

  return <div className="docs-content">{elements}</div>;
}

/* ═══════════════════════════════════════════════════════════════════
   CLI COMMAND CONTENT
   ═══════════════════════════════════════════════════════════════════ */

function CliGroupContent({
  group,
  onHeadings,
}: {
  group: CommandGroup;
  onHeadings?: (headings: TocHeading[]) => void;
}) {
  const headings: TocHeading[] = [
    { id: "commands", text: "Commands", level: 2 },
  ];

  useEffect(() => {
    if (onHeadings) {
      onHeadings(headings);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [group.id]);

  return (
    <div>
      <p className="text-sm text-muted-foreground mb-6">{group.description}</p>
      {group.alias && (
        <p className="text-sm text-muted-foreground mb-4">
          Alias:{" "}
          <code className="rounded bg-muted px-1.5 py-0.5 text-[0.85em] font-mono text-primary">
            {group.alias}
          </code>
        </p>
      )}
      <h2
        id="commands"
        className="text-xl font-bold tracking-tight mt-8 mb-4 text-foreground scroll-mt-20"
      >
        Commands
      </h2>
      <div className="space-y-4">
        {group.commands.map((cmd) => (
          <CliCommandCard key={cmd.name} cmd={cmd} />
        ))}
      </div>
    </div>
  );
}

function CliCommandContent({
  cmd,
  onHeadings,
}: {
  cmd: CliCommand;
  onHeadings?: (headings: TocHeading[]) => void;
}) {
  const headings: TocHeading[] = [];
  if (cmd.usage) headings.push({ id: "usage", text: "Usage", level: 2 });
  if (cmd.options) headings.push({ id: "options", text: "Options", level: 2 });
  if (cmd.subcommands.length > 0)
    headings.push({ id: "see-also", text: "See Also", level: 2 });

  useEffect(() => {
    if (onHeadings) {
      onHeadings(headings);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cmd.name]);

  return (
    <div>
      <p className="text-sm text-muted-foreground mb-6">{cmd.description}</p>
      {cmd.synopsis && (
        <p className="text-sm text-muted-foreground mb-4">{cmd.synopsis}</p>
      )}
      {cmd.usage && (
        <>
          <h2
            id="usage"
            className="text-xl font-bold tracking-tight mt-8 mb-4 text-foreground scroll-mt-20"
          >
            Usage
          </h2>
          <CodeBlock code={cmd.usage} language="bash" />
        </>
      )}
      {cmd.options && (
        <>
          <h2
            id="options"
            className="text-xl font-bold tracking-tight mt-8 mb-4 text-foreground scroll-mt-20"
          >
            Options
          </h2>
          <CodeBlock code={cmd.options} />
        </>
      )}
      {cmd.inheritedOptions && (
        <>
          <h3 className="text-base font-bold tracking-tight mt-6 mb-3 text-foreground">
            Inherited Options
          </h3>
          <CodeBlock code={cmd.inheritedOptions} />
        </>
      )}
      {cmd.subcommands.length > 0 && (
        <>
          <h2
            id="see-also"
            className="text-xl font-bold tracking-tight mt-8 mb-4 text-foreground scroll-mt-20"
          >
            See Also
          </h2>
          <ul className="list-disc pl-5 space-y-1.5">
            {cmd.subcommands.map((sub) => (
              <li
                key={sub.name}
                className="text-sm text-muted-foreground leading-relaxed"
              >
                <code className="rounded bg-muted px-1.5 py-0.5 text-[0.85em] font-mono text-primary">
                  {sub.name}
                </code>{" "}
                — {sub.description}
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

function CliCommandCard({ cmd }: { cmd: CliCommand }) {
  const [expanded, setExpanded] = useState(false);
  const meaningfulOptions = cmd.options
    .split("\n")
    .filter((l) => !l.match(/^\s*-h,\s+--help/) && l.trim())
    .join("\n")
    .trim();

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-accent/30 transition-colors"
        aria-expanded={expanded}
      >
        <div className="flex items-center gap-3">
          <code className="text-sm font-mono font-semibold text-foreground">
            {cmd.name}
          </code>
          <span className="text-xs text-muted-foreground">
            {cmd.description}
          </span>
        </div>
        <ChevronDown
          className={`h-4 w-4 text-muted-foreground transition-transform ${expanded ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
      </button>
      {expanded && (
        <div className="border-t border-border px-4 py-3 bg-accent/10">
          {cmd.usage && (
            <CodeBlock code={cmd.usage} language="bash" title="Usage" />
          )}
          {meaningfulOptions && (
            <div className="mt-2">
              <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                Options
              </div>
              <pre className="text-[12px] font-mono text-muted-foreground bg-[#0C0A08] rounded-lg px-3 py-2 overflow-x-auto">
                <code>{meaningfulOptions}</code>
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   PLACEHOLDER CONTENT
   ═══════════════════════════════════════════════════════════════════ */

function PlaceholderContent({ label }: { label: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Construction
        className="h-12 w-12 text-muted-foreground/30 mb-4"
        aria-hidden="true"
      />
      <h2 className="text-lg font-semibold text-foreground mb-2">{label}</h2>
      <p className="text-sm text-muted-foreground max-w-md">
        This page is under construction. Check back soon or contribute on{" "}
        <a
          href="https://github.com/rpuneet/bc"
          className="text-primary hover:underline"
          target="_blank"
          rel="noopener noreferrer"
        >
          GitHub
        </a>
        .
      </p>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   TABLE OF CONTENTS (right sidebar)
   ═══════════════════════════════════════════════════════════════════ */

function TableOfContents({ headings }: { headings: TocHeading[] }) {
  const [activeId, setActiveId] = useState<string | null>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        }
      },
      { rootMargin: "-80px 0px -70% 0px" },
    );

    for (const h of headings) {
      const el = document.getElementById(h.id);
      if (el) observer.observe(el);
    }

    return () => observer.disconnect();
  }, [headings]);

  if (headings.length === 0) return null;

  return (
    <nav aria-label="Table of contents" className="text-xs">
      <div className="font-semibold text-foreground mb-3 text-xs uppercase tracking-wider">
        On this page
      </div>
      <ul className="space-y-1.5">
        {headings.map((h) => (
          <li key={h.id} style={{ paddingLeft: `${(h.level - 2) * 12}px` }}>
            <a
              href={`#${h.id}`}
              className={`block py-0.5 transition-colors ${
                activeId === h.id
                  ? "text-primary font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {h.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   SIDEBAR NAV SECTION
   ═══════════════════════════════════════════════════════════════════ */

function SidebarSection({
  section,
  activeItemId,
  onItemClick,
  defaultOpen,
}: {
  section: NavSection;
  activeItemId: string | null;
  onItemClick: (item: NavItem) => void;
  defaultOpen: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const Icon = section.icon;
  const isActive = section.items.some((item) => item.id === activeItemId);

  return (
    <div className="mb-1">
      <button
        onClick={() => setOpen(!open)}
        className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm rounded-md transition-colors ${
          isActive
            ? "text-foreground font-medium"
            : "text-muted-foreground hover:text-foreground"
        }`}
        aria-expanded={open}
      >
        {open ? (
          <ChevronDown className="h-3 w-3 shrink-0" aria-hidden="true" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0" aria-hidden="true" />
        )}
        <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        <span className="font-medium">{section.label}</span>
      </button>
      {open && (
        <div className="ml-4 pl-3 border-l border-border/50 mt-0.5">
          {section.items.map((item) => (
            <button
              key={item.id}
              onClick={() => onItemClick(item)}
              className={`block w-full text-left px-3 py-1.5 text-[13px] rounded-md transition-colors ${
                activeItemId === item.id
                  ? "text-primary bg-primary/5 border-l-2 border-primary -ml-[1px] font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   PREVIOUS / NEXT NAVIGATION
   ═══════════════════════════════════════════════════════════════════ */

function PrevNextNav({
  allItems,
  activeItemId,
  onNavigate,
}: {
  allItems: NavItem[];
  activeItemId: string | null;
  onNavigate: (item: NavItem) => void;
}) {
  const currentIdx = allItems.findIndex((item) => item.id === activeItemId);
  const prev = currentIdx > 0 ? allItems[currentIdx - 1] : null;
  const next =
    currentIdx < allItems.length - 1 ? allItems[currentIdx + 1] : null;

  if (!prev && !next) return null;

  return (
    <div className="flex items-center justify-between mt-12 pt-6 border-t border-border">
      {prev ? (
        <button
          onClick={() => onNavigate(prev)}
          className="group flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ChevronRight
            className="h-4 w-4 rotate-180 group-hover:-translate-x-0.5 transition-transform"
            aria-hidden="true"
          />
          <div className="text-left">
            <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70">
              Previous
            </div>
            <div className="font-medium">{prev.label}</div>
          </div>
        </button>
      ) : (
        <div />
      )}
      {next ? (
        <button
          onClick={() => onNavigate(next)}
          className="group flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors text-right"
        >
          <div>
            <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70">
              Next
            </div>
            <div className="font-medium">{next.label}</div>
          </div>
          <ChevronRight
            className="h-4 w-4 group-hover:translate-x-0.5 transition-transform"
            aria-hidden="true"
          />
        </button>
      ) : (
        <div />
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   MAIN DOCS CONTENT
   ═══════════════════════════════════════════════════════════════════ */

export default function DocsContent({
  groups,
  standalone,
  sections,
}: {
  groups: CommandGroup[];
  standalone: CliCommand[];
  sections: DocsSection[];
}) {
  const [tocHeadings, setTocHeadings] = useState<TocHeading[]>([]);
  const [search, setSearch] = useState("");
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const initializedRef = useRef(false);

  // Build navigation structure
  const navSections: NavSection[] = useMemo(() => {
    const result: NavSection[] = [];

    // Getting Started / Tutorials
    const tutorials = sections.find((s) => s.id === "tutorials");
    if (tutorials && tutorials.articles.length > 0) {
      result.push({
        id: "tutorials",
        label: "Getting Started",
        icon: BookOpen,
        items: tutorials.articles.map((a) => ({
          id: `tutorials/${a.slug}`,
          label: a.title,
          type: "article" as const,
          sectionId: "tutorials",
          content: a.content,
          description: a.description,
        })),
      });
    }

    // How-To Guides
    const howto = sections.find((s) => s.id === "how-to");
    if (howto && howto.articles.length > 0) {
      result.push({
        id: "how-to",
        label: "How-To Guides",
        icon: Compass,
        items: howto.articles.map((a) => ({
          id: `how-to/${a.slug}`,
          label: a.title,
          type: "article" as const,
          sectionId: "how-to",
          content: a.content,
          description: a.description,
        })),
      });
    }

    // CLI Reference
    const cliItems: NavItem[] = [];
    for (const group of groups) {
      cliItems.push({
        id: `cli/${group.name}`,
        label: `bc ${group.name}`,
        type: "cli-group",
        sectionId: "cli",
        cliGroup: group,
      });
    }
    // Add standalone commands
    for (const cmd of standalone) {
      cliItems.push({
        id: `cli/${cmd.name.replace(/\s+/g, "-")}`,
        label: cmd.name,
        type: "cli-command",
        sectionId: "cli",
        cliCommand: cmd,
      });
    }
    if (cliItems.length > 0) {
      result.push({
        id: "cli",
        label: "CLI Reference",
        icon: Terminal,
        items: cliItems,
      });
    }

    // API Reference
    const reference = sections.find((s) => s.id === "reference");
    if (reference && reference.articles.length > 0) {
      result.push({
        id: "reference",
        label: "API Reference",
        icon: FileText,
        items: reference.articles.map((a) => ({
          id: `reference/${a.slug}`,
          label: a.title,
          type: "article" as const,
          sectionId: "reference",
          content: a.content,
          description: a.description,
        })),
      });
    }

    // Architecture / Explanation
    const explanation = sections.find((s) => s.id === "explanation");
    if (explanation && explanation.articles.length > 0) {
      result.push({
        id: "explanation",
        label: "Architecture",
        icon: Lightbulb,
        items: explanation.articles.map((a) => ({
          id: `explanation/${a.slug}`,
          label: a.title,
          type: "article" as const,
          sectionId: "explanation",
          content: a.content,
          description: a.description,
        })),
      });
    }

    // Placeholder sections
    result.push({
      id: "contributing",
      label: "Contributing",
      icon: FileText,
      items: [
        {
          id: "contributing/guide",
          label: "Contributing Guide",
          type: "placeholder" as const,
          sectionId: "contributing",
        },
      ],
    });

    result.push({
      id: "release-notes",
      label: "Release Notes",
      icon: FileText,
      items: [
        {
          id: "release-notes/latest",
          label: "Latest Release",
          type: "placeholder" as const,
          sectionId: "release-notes",
        },
      ],
    });

    return result;
  }, [sections, groups, standalone]);

  // Flatten all items for prev/next nav
  const allItems = useMemo(() => {
    return navSections.flatMap((s) => s.items);
  }, [navSections]);

  // Compute initial active item from URL hash or default to first item
  const defaultItemId = useMemo(() => {
    if (typeof window === "undefined" || allItems.length === 0) {
      return allItems[0]?.id ?? null;
    }
    const hash = window.location.hash.slice(1);
    if (hash) {
      const matchingItem = allItems.find(
        (item) => item.id === hash || item.id === decodeURIComponent(hash),
      );
      if (matchingItem) return matchingItem.id;
    }
    return allItems[0]?.id ?? null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [activeItemId, setActiveItemId] = useState<string | null>(
    defaultItemId,
  );

  // On first render in the browser, check hash if SSR couldn't
  if (!initializedRef.current && activeItemId === null && allItems.length > 0) {
    initializedRef.current = true;
    const hash =
      typeof window !== "undefined" ? window.location.hash.slice(1) : "";
    if (hash) {
      const matchingItem = allItems.find(
        (item) => item.id === hash || item.id === decodeURIComponent(hash),
      );
      if (matchingItem) {
        setActiveItemId(matchingItem.id);
      } else {
        setActiveItemId(allItems[0].id);
      }
    } else {
      setActiveItemId(allItems[0].id);
    }
  }

  // Update URL hash when active item changes
  useEffect(() => {
    if (activeItemId) {
      window.history.replaceState(null, "", `#${activeItemId}`);
    }
  }, [activeItemId]);

  // Find active item
  const activeItem = useMemo(() => {
    return allItems.find((item) => item.id === activeItemId) ?? null;
  }, [allItems, activeItemId]);

  // Search filtering
  const filteredSections = useMemo(() => {
    if (!search.trim()) return navSections;
    const q = search.toLowerCase();
    return navSections
      .map((section) => ({
        ...section,
        items: section.items.filter(
          (item) =>
            item.label.toLowerCase().includes(q) ||
            item.description?.toLowerCase().includes(q) ||
            item.sectionId.toLowerCase().includes(q),
        ),
      }))
      .filter((section) => section.items.length > 0);
  }, [navSections, search]);

  const handleItemClick = useCallback(
    (item: NavItem) => {
      setActiveItemId(item.id);
      setMobileNavOpen(false);
      // Scroll content to top
      const contentEl = document.getElementById("docs-content-area");
      if (contentEl) {
        contentEl.scrollTo({ top: 0, behavior: "smooth" });
      }
    },
    [],
  );

  const handleHeadings = useCallback((headings: TocHeading[]) => {
    setTocHeadings(headings);
  }, []);

  // Render active content
  const renderContent = () => {
    if (!activeItem) {
      return (
        <div className="text-center py-16 text-muted-foreground">
          <p>Select a page from the sidebar.</p>
        </div>
      );
    }

    if (activeItem.type === "placeholder") {
      return <PlaceholderContent label={activeItem.label} />;
    }

    if (activeItem.type === "cli-group" && activeItem.cliGroup) {
      return (
        <CliGroupContent
          group={activeItem.cliGroup}
          onHeadings={handleHeadings}
        />
      );
    }

    if (activeItem.type === "cli-command" && activeItem.cliCommand) {
      return (
        <CliCommandContent
          cmd={activeItem.cliCommand}
          onHeadings={handleHeadings}
        />
      );
    }

    if (activeItem.type === "article" && activeItem.content) {
      return (
        <MarkdownContent
          content={activeItem.content}
          onHeadings={handleHeadings}
        />
      );
    }

    return <PlaceholderContent label={activeItem.label} />;
  };

  // Get title for active item
  const getTitle = () => {
    if (!activeItem) return "Documentation";
    if (activeItem.type === "cli-group" && activeItem.cliGroup) {
      return `bc ${activeItem.cliGroup.name}`;
    }
    if (activeItem.type === "cli-command" && activeItem.cliCommand) {
      return activeItem.cliCommand.name;
    }
    return activeItem.label;
  };

  const getDescription = () => {
    if (!activeItem) return "";
    if (activeItem.type === "cli-group" && activeItem.cliGroup) {
      return activeItem.cliGroup.description;
    }
    if (activeItem.type === "cli-command" && activeItem.cliCommand) {
      return activeItem.cliCommand.description;
    }
    return activeItem.description || "";
  };

  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground overflow-x-hidden">
      <Nav />

      {/* Mobile nav toggle */}
      <div className="lg:hidden sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur-sm px-4 py-3 flex items-center gap-3">
        <button
          onClick={() => setMobileNavOpen(!mobileNavOpen)}
          className="p-2 rounded-md hover:bg-accent/30 transition-colors"
          aria-label="Toggle navigation"
        >
          {mobileNavOpen ? (
            <X className="h-5 w-5" />
          ) : (
            <Menu className="h-5 w-5" />
          )}
        </button>
        <span className="text-sm font-medium truncate">
          {getTitle()}
        </span>
      </div>

      <div className="flex min-h-[calc(100vh-64px)]">
        {/* ═══ LEFT SIDEBAR ═══ */}
        <aside
          className={`${
            mobileNavOpen
              ? "fixed inset-0 top-[105px] z-30 bg-background"
              : "hidden"
          } lg:block lg:sticky lg:top-0 lg:h-screen w-full lg:w-[260px] xl:w-[280px] shrink-0 border-r border-border overflow-y-auto`}
        >
          <div className="p-4 pt-20 lg:pt-6">
            {/* Search */}
            <div className="relative mb-5">
              <Search
                className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground"
                aria-hidden="true"
              />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search docs..."
                className="h-9 w-full rounded-lg border border-border bg-card pl-9 pr-3 text-sm outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary/50 focus:ring-1 focus:ring-primary/10"
              />
            </div>

            {/* Nav sections */}
            <nav aria-label="Documentation navigation">
              {filteredSections.map((section) => (
                <SidebarSection
                  key={section.id}
                  section={section}
                  activeItemId={activeItemId}
                  onItemClick={handleItemClick}
                  defaultOpen={
                    section.id === "tutorials" ||
                    section.items.some((item) => item.id === activeItemId)
                  }
                />
              ))}
              {search && filteredSections.length === 0 && (
                <div className="text-center py-8 text-muted-foreground">
                  <Search
                    className="h-6 w-6 mx-auto mb-2 opacity-30"
                    aria-hidden="true"
                  />
                  <p className="text-xs">No results for &quot;{search}&quot;</p>
                  <button
                    onClick={() => setSearch("")}
                    className="mt-1 text-xs text-primary hover:underline"
                  >
                    Clear search
                  </button>
                </div>
              )}
            </nav>
          </div>
        </aside>

        {/* ═══ MAIN CONTENT ═══ */}
        <div
          id="docs-content-area"
          className="flex-1 min-w-0 overflow-y-auto"
        >
          <div className="max-w-3xl mx-auto px-6 lg:px-10 py-8 lg:py-12">
            {/* Breadcrumb */}
            {activeItem && (
              <div className="text-xs text-muted-foreground mb-4 flex items-center gap-1.5">
                <span>Docs</span>
                <ChevronRight className="h-3 w-3" aria-hidden="true" />
                <span>
                  {navSections.find((s) =>
                    s.items.some((i) => i.id === activeItemId),
                  )?.label || ""}
                </span>
                <ChevronRight className="h-3 w-3" aria-hidden="true" />
                <span className="text-foreground">{getTitle()}</span>
              </div>
            )}

            {/* Title */}
            <h1 className="text-3xl lg:text-4xl font-bold tracking-tight mb-2 text-foreground">
              {getTitle()}
            </h1>
            {getDescription() && (
              <p className="text-base text-muted-foreground mb-8 leading-relaxed">
                {getDescription()}
              </p>
            )}

            <div className="border-t border-border pt-6">
              {renderContent()}
            </div>

            {/* Previous / Next */}
            <PrevNextNav
              allItems={allItems}
              activeItemId={activeItemId}
              onNavigate={handleItemClick}
            />
          </div>
        </div>

        {/* ═══ RIGHT SIDEBAR (Table of Contents) ═══ */}
        <aside className="hidden xl:block sticky top-0 h-screen w-[200px] shrink-0 overflow-y-auto border-l border-border">
          <div className="p-4 pt-20 lg:pt-12">
            <TableOfContents headings={tocHeadings} />
          </div>
        </aside>
      </div>

      <Footer />
    </main>
  );
}
