import { Metadata } from "next";
import { BreadcrumbSchema } from "../_components/StructuredData";

export const metadata: Metadata = {
  title: "Changelog — bc | Release Notes & Updates",
  description:
    "What's new in bc. Release notes, features, and improvements for the CLI-first multi-agent orchestration platform.",
  alternates: {
    canonical: "/changelog",
  },
  openGraph: {
    title: "Changelog — bc | Release Notes & Updates",
    description:
      "What's new in bc. Release notes, features, and improvements for the CLI-first multi-agent orchestration platform.",
    url: "https://bc-infra.com/changelog",
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
    title: "Changelog — bc | Release Notes & Updates",
    description:
      "What's new in bc. Release notes, features, and improvements for the CLI-first multi-agent orchestration platform.",
    images: ["https://bc-infra.com/og-image.png"],
    creator: "@bcinfra",
  },
};

export default function ChangelogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      {BreadcrumbSchema([
        { name: "Home", url: "https://bc-infra.com" },
        { name: "Changelog", url: "https://bc-infra.com/changelog" },
      ])}
      {children}
    </>
  );
}
