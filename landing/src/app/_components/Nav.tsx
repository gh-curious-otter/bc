"use client";

import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { useState, useRef, useEffect } from "react";
import { Menu, X, Download, Copy, Check } from "lucide-react";
import { ThemeToggle } from "./ThemeToggle";

const links = [
  { href: "/", label: "Home" },
  { href: "/product", label: "Product" },
  { href: "/docs", label: "Docs" },
  { href: "/waitlist", label: "Waitlist" },
];

function Logo() {
  return (
    <div className="flex items-center group">
      <span className="font-mono text-lg font-normal text-secondary/70">&gt;</span>
      <span className="font-heading text-xl font-bold tracking-tight text-primary ml-1">bc</span>
    </div>
  );
}

function HamburgerButton({ isOpen, onClick }: { isOpen: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="p-1.5 rounded-md hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
      aria-label={isOpen ? "Close menu" : "Open menu"}
      aria-expanded={isOpen}
      aria-controls="mobile-menu"
    >
      <motion.div
        animate={isOpen ? "open" : "closed"}
        variants={{
          open: { rotate: 90 },
          closed: { rotate: 0 },
        }}
        transition={{ duration: 0.2 }}
      >
        {isOpen ? <X size={20} aria-hidden="true" /> : <Menu size={20} aria-hidden="true" />}
      </motion.div>
    </button>
  );
}

function InstallDropdown() {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  const commands = [
    { label: "Homebrew", cmd: "brew install bcinfra1/tap/bc" },
    { label: "Go Install", cmd: "go install github.com/bcinfra1/bc@latest" },
    { label: "Binary", cmd: "curl -fsSL https://bc-infra.com/install | sh" },
  ];

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    if (open) document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  function copy(cmd: string) {
    navigator.clipboard.writeText(cmd);
    setCopied(cmd);
    setTimeout(() => setCopied(null), 2000);
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3.5 py-1 text-[13px] font-medium text-primary-foreground transition-all hover:opacity-90 active:scale-95 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary whitespace-nowrap"
      >
        <Download className="h-3 w-3" aria-hidden="true" />
        Install
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.95 }}
            transition={{ duration: 0.15 }}
            className="absolute right-0 top-full mt-2 w-80 rounded-lg border border-border bg-card shadow-xl overflow-hidden z-50"
          >
            <div className="px-3 py-2 border-b border-border/60">
              <span className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">Quick install</span>
            </div>
            {commands.map((c) => (
              <div key={c.label} className="px-3 py-2.5 flex items-center gap-2 hover:bg-accent/30 transition-colors group">
                <div className="flex-1 min-w-0">
                  <div className="text-[10px] font-medium text-muted-foreground mb-0.5">{c.label}</div>
                  <code className="text-xs font-mono text-foreground block truncate">{c.cmd}</code>
                </div>
                <button
                  onClick={() => copy(c.cmd)}
                  className="shrink-0 p-1 rounded hover:bg-accent transition-colors"
                  aria-label={`Copy ${c.label} command`}
                >
                  {copied === c.cmd ? (
                    <Check className="h-3.5 w-3.5 text-success" />
                  ) : (
                    <Copy className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                  )}
                </button>
              </div>
            ))}
            <div className="px-3 py-2 border-t border-border/60 bg-muted/30">
              <Link
                href="/docs#installation"
                onClick={() => setOpen(false)}
                className="text-[11px] text-muted-foreground hover:text-foreground transition-colors"
              >
                Full installation guide →
              </Link>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export function Nav() {
  const [isOpen, setIsOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onScroll() {
      setScrolled(window.scrollY > 10);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    if (isOpen) document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen]);

  useEffect(() => {
    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") setIsOpen(false);
    }
    if (isOpen) document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [isOpen]);

  const handleLinkClick = () => setIsOpen(false);

  return (
    <header
      className={`sticky top-0 z-50 transition-all duration-300 ${
        scrolled
          ? "border-b border-border/50 bg-background/75 backdrop-blur-xl shadow-sm"
          : "border-b border-transparent bg-transparent"
      }`}
    >
      <div className="mx-auto flex max-w-6xl items-center px-4 py-3 sm:px-6">
        {/* Left: Logo + Nav links */}
        <div className="flex items-center gap-1">
          <Link href="/" className="rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary" aria-label="bc home page">
            <Logo />
          </Link>
          <div className="hidden md:block w-[2px] h-4 bg-primary/60 mx-2 animate-[blink_1s_step-end_infinite]" />
          <nav aria-label="Main navigation" className="hidden items-center md:flex">
            {links.map((l) => (
              <Link
                key={l.href}
                href={l.href}
                className="rounded-md px-2.5 py-1 text-[13px] font-medium text-muted-foreground transition-colors hover:text-foreground hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                {l.label}
              </Link>
            ))}
          </nav>
        </div>

        {/* Right: Theme toggle + Install */}
        <div className="hidden md:flex items-center gap-2 ml-auto">
          <ThemeToggle />
          <InstallDropdown />
        </div>

        {/* Mobile: hamburger */}
        <div className="md:hidden ml-auto">
          <HamburgerButton isOpen={isOpen} onClick={() => setIsOpen(!isOpen)} />
        </div>
      </div>

      {/* Mobile Menu Panel */}
      <AnimatePresence>
        {isOpen && (
          <motion.div
            ref={menuRef}
            id="mobile-menu"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2, ease: "easeInOut" }}
            className="md:hidden border-t border-border/50 bg-background/95 backdrop-blur-sm"
          >
            <nav aria-label="Mobile navigation" className="flex flex-col px-4 py-2 space-y-0.5">
              {links.map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  onClick={handleLinkClick}
                  className="rounded-md px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground transition-colors flex items-center"
                >
                  {l.label}
                </Link>
              ))}
              <div className="h-px bg-border/40 my-1" />
              <div className="px-3 py-2">
                <div className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground mb-2">Install</div>
                <code className="block text-xs font-mono text-foreground bg-muted/50 rounded px-2.5 py-2 mb-1.5">
                  brew install bcinfra1/tap/bc
                </code>
                <Link
                  href="/docs#installation"
                  onClick={handleLinkClick}
                  className="text-[11px] text-muted-foreground hover:text-foreground transition-colors"
                >
                  More options →
                </Link>
              </div>
              <div className="h-px bg-border/40 my-1" />
              <div className="px-3 py-2 flex items-center justify-between">
                <span className="text-sm font-medium text-muted-foreground">Theme</span>
                <ThemeToggle />
              </div>
            </nav>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
