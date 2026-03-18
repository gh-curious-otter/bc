import { Metadata } from "next";
import { BreadcrumbSchema } from "../_components/StructuredData";

export const metadata: Metadata = {
  title: "Documentation — bc | Quick Start, CLI Reference & Guides",
  description:
    "Complete bc documentation: installation, quick start, all 55 CLI commands, configuration, presets, and environment variables. CLI-first multi-agent orchestration.",
  alternates: {
    canonical: "/docs",
  },
  openGraph: {
    title: "bc Documentation — Quick Start, CLI Reference & Guides",
    description:
      "Complete bc documentation: installation, quick start, all 55 CLI commands, configuration, presets, and environment variables.",
    url: "https://bc-infra.com/docs",
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
    title: "bc Documentation — Quick Start, CLI Reference & Guides",
    description:
      "Complete bc documentation: installation, quick start, all 55 CLI commands, configuration, presets, and environment variables.",
    images: ["https://bc-infra.com/og-image.png"],
    creator: "@bcinfra",
  },
};

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      {BreadcrumbSchema([
        { name: "Home", url: "https://bc-infra.com" },
        { name: "Documentation", url: "https://bc-infra.com/docs" },
      ])}
      {children}
    </>
  );
}
