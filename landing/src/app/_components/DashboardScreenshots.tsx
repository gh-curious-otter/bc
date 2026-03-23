"use client";

import { useState } from "react";
import Image from "next/image";
import { motion, AnimatePresence } from "framer-motion";

const TABS = [
  {
    id: "dashboard",
    label: "Dashboard",
    src: "/screenshots/dashboard-01-home.png",
    alt: "bc dashboard showing active agents, channels, total cost, and token usage",
  },
  {
    id: "agents",
    label: "Agents",
    src: "/screenshots/dashboard-02-agents.png",
    alt: "bc agents table showing agent names, roles, tools, tasks, and statuses",
  },
  {
    id: "channels",
    label: "Channels",
    src: "/screenshots/dashboard-03-channels.png",
    alt: "bc channel view showing real-time agent-to-agent communication",
  },
  {
    id: "costs",
    label: "Costs",
    src: "/screenshots/dashboard-04-costs.png",
    alt: "bc cost tracking with daily trend chart and per-agent cost breakdown",
  },
  {
    id: "stats",
    label: "Stats",
    src: "/screenshots/dashboard-10-stats-loaded.png",
    alt: "bc stats overview showing system metrics and agent performance",
  },
] as const;

export function DashboardScreenshots() {
  const [active, setActive] = useState(0);

  return (
    <div className="flex flex-col gap-5">
      {/* Browser chrome */}
      <div className="overflow-hidden rounded-xl border border-border shadow-[0_20px_60px_-15px_rgba(0,0,0,0.3)] dark:shadow-[0_20px_60px_-15px_rgba(0,0,0,0.5)] bg-card">
        {/* Address bar */}
        <div className="flex items-center gap-3 border-b border-border bg-muted/30 dark:bg-[#151210] px-4 py-2.5">
          <div className="flex gap-1.5" aria-hidden="true">
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--traffic-red)]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--traffic-yellow)]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--traffic-green)]" />
          </div>
          <div className="flex-1 flex justify-center">
            <div className="inline-flex items-center gap-2 rounded-md bg-background/60 dark:bg-[#0c0a08]/60 border border-border/50 px-4 py-1 text-xs text-muted-foreground font-mono">
              <svg className="h-3 w-3 text-success" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M8 0a8 8 0 1 0 0 16A8 8 0 0 0 8 0ZM4.5 7.5a.5.5 0 0 0 0 1h4.793l-2.147 2.146a.5.5 0 0 0 .708.708l3-3a.5.5 0 0 0 0-.708l-3-3a.5.5 0 1 0-.708.708L9.293 7.5H4.5Z" clipRule="evenodd"/>
              </svg>
              localhost:9374
            </div>
          </div>
        </div>

        {/* Tab bar */}
        <div className="flex items-center gap-0.5 border-b border-border bg-muted/20 dark:bg-[#121010] px-4 py-1">
          {TABS.map((tab, i) => (
            <button
              key={tab.id}
              onClick={() => setActive(i)}
              className={`relative rounded-md px-4 py-2 text-sm font-medium transition-all duration-200 ${
                i === active
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/20"
              }`}
              aria-label={`View ${tab.label} screenshot`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Screenshot with crossfade */}
        <div className="relative overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.div
              key={TABS[active].id}
              initial={{ opacity: 0, scale: 1.01 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.99 }}
              transition={{ duration: 0.3, ease: "easeInOut" }}
            >
              <Image
                src={TABS[active].src}
                alt={TABS[active].alt}
                width={1200}
                height={750}
                className="w-full h-auto"
                priority={active === 0}
              />
            </motion.div>
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
