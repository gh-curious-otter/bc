"use client";

import { motion } from "framer-motion";

export const FadeIn = ({
  children,
  delay = 0,
  className = "",
}: {
  children: React.ReactNode;
  delay?: number;
  className?: string;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 14 }}
    whileInView={{ opacity: 1, y: 0 }}
    viewport={{ once: true, margin: "-80px" }}
    transition={{ duration: 0.6, ease: "easeOut", delay }}
    className={className}
    style={{ willChange: "transform, opacity" }}
  >
    {children}
  </motion.div>
);

export const Floaty = ({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) => (
  <motion.div
    animate={{ y: [0, -6, 0] }}
    transition={{ duration: 4.5, repeat: Infinity, ease: "easeInOut" }}
    className={className}
    style={{ willChange: "transform" }}
  >
    {children}
  </motion.div>
);
