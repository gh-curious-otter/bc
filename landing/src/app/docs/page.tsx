import { loadCliDocs } from "@/lib/cli-docs";
import { loadAllDocs } from "@/lib/docs-loader";
import DocsContent from "./DocsContent";

export default function DocsPage() {
  const { groups, standalone } = loadCliDocs();
  const sections = loadAllDocs();
  return (
    <DocsContent groups={groups} standalone={standalone} sections={sections} />
  );
}
