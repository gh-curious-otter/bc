"use client";

import { motion } from "framer-motion";
import { Layers, Code2, Monitor, Github } from "lucide-react";

interface StatItemProps {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  value: string;
  label: string;
  delay?: number;
}

function StatItem({ icon: Icon, value, label, delay = 0 }: StatItemProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay, duration: 0.5, ease: "easeOut" }}
      className="flex flex-col items-center gap-2.5 p-5 rounded-xl border border-border/50 bg-card/40 backdrop-blur-sm transition-all duration-300 hover:border-primary/20 hover:bg-card/60"
    >
      <div className="flex items-center gap-2 text-primary/60">
        <Icon size={18} aria-hidden="true" />
      </div>
      <span className="text-3xl font-bold tracking-tight font-heading">{value}</span>
      <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground font-bold">
        {label}
      </span>
    </motion.div>
  );
}

export function StatsBar() {
  return (
    <div className="w-full py-10 border-y border-border/40">
      <div className="mx-auto max-w-4xl px-6">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <StatItem icon={Layers} value="7" label="AI Tools" delay={0} />
          <StatItem icon={Github} value="OSS" label="Open Source" delay={0.1} />
          <StatItem
            icon={Code2}
            value="Local"
            label="Runs on your machine"
            delay={0.2}
          />
          <StatItem
            icon={Monitor}
            value="Free"
            label="No login required"
            delay={0.3}
          />
        </div>
      </div>
    </div>
  );
}
