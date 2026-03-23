import { Metadata } from "next";
import { BreadcrumbSchema } from "../_components/StructuredData";

export const metadata: Metadata = {
  title: "Now — bc | What We're Working On",
  description:
    "What's happening with bc right now. Current focus, recent updates, and what's next.",
  alternates: {
    canonical: "/now",
  },
  openGraph: {
    title: "Now — bc | What We're Working On",
    description:
      "What's happening with bc right now. Current focus, recent updates, and what's next.",
    url: "https://bc-infra.com/now",
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
    title: "Now — bc | What We're Working On",
    description:
      "What's happening with bc right now. Current focus, recent updates, and what's next.",
    images: ["https://bc-infra.com/og-image.png"],
    creator: "@bcinfra",
  },
};

export default function NowLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      {BreadcrumbSchema([
        { name: "Home", url: "https://bc-infra.com" },
        { name: "Now", url: "https://bc-infra.com/now" },
      ])}
      {children}
    </>
  );
}
