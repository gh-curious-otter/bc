import { loadCliDocs } from "@/lib/cli-docs";
import DocsContent from "./DocsContent";

export default function DocsPage() {
  const { groups, standalone } = loadCliDocs();
  return <DocsContent groups={groups} standalone={standalone} />;
}
