import { Metadata } from "next";
import { BreadcrumbSchema, FAQSchema } from "../_components/StructuredData";

export const metadata: Metadata = {
  title: "Waitlist — bc | Early Access to Multi-Agent Orchestration",
  description:
    "Join the bc waitlist and get early access to the CLI-first multi-agent orchestration platform. Run 5+ AI coding agents simultaneously with zero conflicts.",
  alternates: {
    canonical: "/waitlist",
  },
  openGraph: {
    title: "bc Waitlist — Early Access to Multi-Agent Orchestration",
    description:
      "Get early access to bc's CLI-first multi-agent orchestration platform. Run 5+ AI coding agents with zero conflicts.",
    url: "https://bc-infra.com/waitlist",
    siteName: "bc",
    type: "website",
    images: [
      {
        url: "https://bc-infra.com/og-image.png",
        width: 1200,
        height: 630,
        alt: "bc - Multi-Agent Orchestration Platform",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "bc Waitlist — Early Access to Multi-Agent Orchestration",
    description:
      "Get early access to bc's CLI-first multi-agent orchestration platform. Run 5+ AI coding agents with zero conflicts.",
    images: ["https://bc-infra.com/og-image.png"],
    creator: "@bcinfra",
  },
};

export default function WaitlistLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      {BreadcrumbSchema([
        { name: "Home", url: "https://bc-infra.com" },
        { name: "Waitlist", url: "https://bc-infra.com/waitlist" },
      ])}
      {FAQSchema([
        {
          question: "What is bc?",
          answer:
            "bc is a CLI-first multi-agent orchestration platform. It coordinates multiple AI coding agents — like Claude Code, Cursor, Codex, Gemini, and others — so they can work in parallel on isolated git worktrees without merge conflicts or context loss.",
        },
        {
          question: "How is this different from using a single AI agent?",
          answer:
            "A single agent works serially on one task at a time. bc lets you run 5-10 agents simultaneously, each on its own branch, communicating through structured channels. Think of it as going from one developer to a full engineering team — with cost controls and real-time visibility.",
        },
        {
          question: "Which AI tools does bc support?",
          answer:
            "bc is agent-agnostic. It works with Claude Code, Cursor, Codex, Gemini, Aider, OpenCode, OpenClaw, and any CLI-based coding assistant. You configure providers in a simple TOML file and bc handles the coordination.",
        },
        {
          question: "Do I need to change how my agents work?",
          answer:
            "No. bc orchestrates agents you already use with zero code changes. Your agents keep running the same commands — bc adds the coordination layer on top: worktree isolation, channels for communication, persistent memory, and cost tracking.",
        },
        {
          question: "Is bc open source?",
          answer:
            "Yes. bc is MIT licensed and open source. You can inspect the code, contribute, and self-host. The beta gives you access to the full platform — CLI, Web UI dashboard, and all features.",
        },
        {
          question: "How does the beta work?",
          answer:
            "Sign up with your email and we will onboard you into the private beta. You will get full CLI access, priority support, and direct input on the product roadmap. Beta users help shape the future of AI coordination.",
        },
      ])}
      {children}
    </>
  );
}
