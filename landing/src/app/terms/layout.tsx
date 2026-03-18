import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Terms of Service — bc",
  description:
    "bc terms of service — usage license, acceptable use policy, limitations of liability, and governing law for the bc multi-agent orchestration platform.",
  alternates: {
    canonical: "/terms",
  },
  robots: {
    index: false,
  },
};

export default function TermsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
