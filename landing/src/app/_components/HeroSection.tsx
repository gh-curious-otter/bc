"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import {
  ArrowRight,
  Terminal,
  CheckCircle2,
  Layers,
  Monitor,
} from "lucide-react";
import { TerminalWindow, CommandOutput } from "./TerminalComponents";

const fadeUp = {
  hidden: { opacity: 0, y: 30 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: i * 0.1, duration: 0.6, ease: "easeOut" as const },
  }),
};

const stagger = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.08 } },
};

export function HeroSection() {
  return (
    <div className="relative mx-auto max-w-7xl px-4 sm:px-6 pb-0 pt-8 lg:pt-20">
      <motion.section
        initial="hidden"
        animate="visible"
        variants={stagger}
        className="grid items-center gap-8 lg:grid-cols-[1.1fr_1fr] lg:gap-16"
      >
        <div className="flex flex-col items-start">
          <motion.div
            variants={fadeUp}
            custom={0}
            className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-accent/10 px-4 py-1.5 font-mono text-xs text-muted-foreground backdrop-blur-sm"
          >
            <span className="h-1.5 w-1.5 rounded-full bg-success animate-pulse" />
            CLI-first &middot; Agent-agnostic &middot; Private beta
          </motion.div>

          <motion.h1
            variants={fadeUp}
            custom={1}
            className="text-balance text-[2.25rem] font-bold leading-[1.05] tracking-tight sm:text-5xl lg:text-7xl"
          >
            Multi-agent
            <br />
            orchestration
            <br />
            <span className="text-muted-foreground/40">
              from your terminal.
            </span>
          </motion.h1>

          <motion.p
            variants={fadeUp}
            custom={2}
            className="mt-4 max-w-[520px] text-base leading-relaxed text-muted-foreground sm:text-lg"
          >
            Coordinate teams of AI coding agents from your terminal &mdash; with
            isolated git worktrees, structured channels, persistent memory, and
            cost controls.
          </motion.p>

          <motion.div
            variants={fadeUp}
            custom={3}
            className="mt-6 flex flex-wrap items-center gap-3"
          >
            <Link
              href="/waitlist"
              className="group inline-flex h-10 sm:h-11 items-center gap-2 rounded-lg bg-primary px-6 sm:px-8 text-sm font-semibold text-primary-foreground shadow-[var(--btn-shadow)] transition-all hover:shadow-xl hover:shadow-primary/20 active:scale-[0.97]"
              aria-label="Join the bc waitlist"
            >
              Request Early Access
              <ArrowRight
                className="h-4 w-4 transition-transform group-hover:translate-x-0.5"
                aria-hidden="true"
              />
            </Link>
            <Link
              href="/docs"
              className="inline-flex h-10 sm:h-11 items-center gap-2 rounded-lg border border-border px-6 sm:px-8 text-sm font-medium transition-colors hover:bg-accent/20 active:scale-[0.97]"
              aria-label="Read the bc documentation"
            >
              <Terminal className="h-4 w-4" aria-hidden="true" />
              View Docs
            </Link>
          </motion.div>

          <motion.div
            variants={fadeUp}
            custom={4}
            className="mt-8 flex flex-wrap items-center gap-6 font-mono text-xs text-muted-foreground"
          >
            <span className="flex items-center gap-1.5">
              <CheckCircle2
                className="h-3.5 w-3.5 text-success"
                aria-hidden="true"
              />
              Open source
            </span>
            <span className="flex items-center gap-1.5">
              <Layers className="h-3.5 w-3.5" aria-hidden="true" />8 AI tools
              supported
            </span>
            <span className="flex items-center gap-1.5">
              <Monitor className="h-3.5 w-3.5" aria-hidden="true" />
              Web UI dashboard
            </span>
          </motion.div>
        </div>

        {/* Hero terminal */}
        <motion.div variants={fadeUp} custom={2} className="relative">
          <div className="absolute -inset-8 rounded-3xl bg-gradient-to-tr from-primary/5 via-transparent to-secondary/10 blur-3xl hero-glow" />
          <div className="relative">
            <TerminalWindow
              title="bc up"
              ariaLabel="Terminal running bc up command, starting 5 AI coding agents in parallel"
            >
              <CommandOutput
                command="bc up"
                lines={[
                  {
                    text: "Starting 5 agents...",
                    color: "text-terminal-muted",
                  },
                  { text: "" },
                  {
                    text: '  \u2713 pm-01       product-manager   working   "Planning sprint"',
                    color: "text-terminal-success",
                  },
                  {
                    text: '  \u2713 mgr-01      manager           working   "Reviewing PRs"',
                    color: "text-terminal-success",
                  },
                  {
                    text: '  \u2713 eng-01      engineer          working   "Building auth"',
                    color: "text-terminal-success",
                  },
                  {
                    text: '  \u2713 eng-02      engineer          working   "Fixing bugs"',
                    color: "text-terminal-success",
                  },
                  {
                    text: '  \u2713 eng-03      engineer          working   "Writing tests"',
                    color: "text-terminal-success",
                  },
                  { text: "" },
                  {
                    text: "All agents active. Dashboard: bc home",
                    color: "text-terminal-muted",
                  },
                ]}
              />
            </TerminalWindow>
          </div>
        </motion.div>
      </motion.section>
    </div>
  );
}
