#!/usr/bin/env python3
"""
Regenerate GitHub issue bodies to use exact agent instructions from templates.

Preserves each issue's existing content but replaces the Agent Instructions
section with the canonical version from the YAML template.
"""

import json
import re
import subprocess
import sys
import yaml
from pathlib import Path

TEMPLATE_DIR = Path(".github/ISSUE_TEMPLATE")


def load_agent_instructions(template_name: str) -> str:
    """Extract the agent instructions value from a YAML template."""
    path = TEMPLATE_DIR / f"{template_name}.yml"
    with open(path) as f:
        tmpl = yaml.safe_load(f)

    for field in tmpl.get("body", []):
        if field.get("id") == "agent-instructions":
            return field["attributes"].get("value", "").strip()
    return ""


def classify_issue(labels: list[str]) -> str:
    """Determine which template an issue should use."""
    if "epic" in labels:
        return "epic"
    if "bug" in labels:
        return "bug_report"
    return "feature_request"


def replace_agent_instructions(body: str, new_instructions: str) -> str:
    """Replace the agent instructions section in an issue body.

    Handles multiple formats:
    - ## Agent Instructions followed by <details> block
    - ### Agent Workflow heading
    - <details><summary>Agent Workflow blocks without a heading
    """
    # Pattern 1: ## Agent Instructions\n\n<details>...</details>
    pattern1 = r'## Agent Instructions\s*\n.*?</details>'
    # Pattern 2: ### Agent Workflow\n<details>...</details>
    pattern2 = r'###?\s*Agent Workflow.*?</details>'
    # Pattern 3: Standalone <details><summary>Agent Workflow...</details>
    pattern3 = r'<details>\s*<summary>Agent Workflow.*?</details>'

    section = f"## Agent Instructions\n\n{new_instructions}"

    for pattern in [pattern1, pattern2, pattern3]:
        if re.search(pattern, body, re.DOTALL):
            return re.sub(pattern, section, body, count=1, flags=re.DOTALL)

    # No existing agent instructions found — append
    return body.rstrip() + "\n\n" + section


def get_open_issues() -> list:
    """Fetch all open issues."""
    result = subprocess.run(
        ["gh", "issue", "list", "--state", "open", "--limit", "100",
         "--json", "number,title,labels,body"],
        capture_output=True, text=True
    )
    return json.loads(result.stdout)


def main():
    issues = get_open_issues()
    print(f"Processing {len(issues)} open issues\n")

    for issue in sorted(issues, key=lambda x: x["number"]):
        num = issue["number"]
        title = issue["title"][:60]
        labels = [l["name"] for l in issue.get("labels", [])]
        body = issue.get("body", "") or ""

        template_name = classify_issue(labels)
        instructions = load_agent_instructions(template_name)

        if not instructions:
            print(f"  #{num:4d}  SKIP — no agent instructions in {template_name} template")
            continue

        new_body = replace_agent_instructions(body, instructions)

        # Check if anything changed
        if new_body == body:
            print(f"  #{num:4d}  UNCHANGED  {title}")
            continue

        # Write preview
        out_path = Path(f"/tmp/issue-{num}.md")
        out_path.write_text(new_body)
        print(f"  #{num:4d}  [{template_name:16s}]  {title}")

    print(f"\nPreview files in /tmp/issue-*.md")
    print("Run with --apply to update GitHub.")

    if "--apply" in sys.argv:
        print("\nApplying updates...")
        for issue in sorted(issues, key=lambda x: x["number"]):
            num = issue["number"]
            out_path = Path(f"/tmp/issue-{num}.md")
            if not out_path.exists():
                continue
            new_body = out_path.read_text()
            old_body = issue.get("body", "") or ""
            if new_body == old_body:
                continue

            result = subprocess.run(
                ["gh", "issue", "edit", str(num), "--body", new_body],
                capture_output=True, text=True
            )
            if result.returncode == 0:
                print(f"  #{num}: updated")
            else:
                print(f"  #{num}: FAILED — {result.stderr.strip()}")


if __name__ == "__main__":
    main()
