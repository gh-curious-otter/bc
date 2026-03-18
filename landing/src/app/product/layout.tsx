import { Metadata } from "next";
import { BreadcrumbSchema } from "../_components/StructuredData";

export const metadata: Metadata = {
  title: "Product — bc | Agent Orchestration, Channels, Costs & More",
  description:
    "Explore bc's multi-agent orchestration platform: agent lifecycle management, inter-agent channels, cost controls, role-based permissions, cron jobs, and more. A CLI-first orchestration platform.",
  alternates: {
    canonical: "/product",
  },
  openGraph: {
    title: "bc Product — Agent Orchestration, Channels, Costs & More",
    description:
      "Explore bc's multi-agent orchestration platform: agent lifecycle, channels, cost controls, roles, and cron jobs. A CLI-first orchestration platform.",
    url: "https://bc-infra.com/product",
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
    title: "bc Product — Agent Orchestration, Channels, Costs & More",
    description:
      "Explore bc's multi-agent orchestration platform: agent lifecycle, channels, cost controls, roles, and cron jobs. A CLI-first orchestration platform.",
    images: ["https://bc-infra.com/og-image.png"],
    creator: "@bcinfra",
  },
};

export default function ProductLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      {BreadcrumbSchema([
        { name: "Home", url: "https://bc-infra.com" },
        { name: "Product", url: "https://bc-infra.com/product" },
      ])}
      {children}
    </>
  );
}
