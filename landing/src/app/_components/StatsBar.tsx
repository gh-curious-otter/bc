"use client";

import { motion } from "framer-motion";
import { GitPullRequest, CheckCircle2, Users, Clock } from "lucide-react";

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
            className="flex flex-col items-center gap-2 p-4"
        >
            <div className="flex items-center gap-2 text-primary/60">
                <Icon size={16} aria-hidden="true" />
            </div>
            <span className="text-2xl font-bold tracking-tight">{value}</span>
            <span className="text-[10px] uppercase tracking-widest text-muted-foreground font-bold">
                {label}
            </span>
        </motion.div>
    );
}

export function StatsBar() {
    return (
        <div className="w-full py-8 border-y border-border/50 bg-accent/20">
            <div className="mx-auto max-w-4xl px-6">
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                    <StatItem
                        icon={GitPullRequest}
                        value="1.2k+"
                        label="PRs Merged"
                        delay={0}
                    />
                    <StatItem
                        icon={CheckCircle2}
                        value="340+"
                        label="Issues Closed"
                        delay={0.1}
                    />
                    <StatItem
                        icon={Users}
                        value="48"
                        label="Active Teams"
                        delay={0.2}
                    />
                    <StatItem
                        icon={Clock}
                        value="10k+"
                        label="Agent Hours"
                        delay={0.3}
                    />
                </div>
            </div>
        </div>
    );
}
