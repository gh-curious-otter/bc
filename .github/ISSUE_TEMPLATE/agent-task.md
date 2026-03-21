---
name: Agent Task
about: Task for AI agents to pick up and execute
labels: ''
---

## GOAL


## CONSTRAINTS

-

## REFERENCES

-

## DONE WHEN

- [ ]
- [ ] `make check` passes
- [ ] PR reviewed and approved

---

<details>
<summary>Agent Instructions</summary>

When you pick up this issue, follow this workflow:

1. **ASSESS SCOPE**
   - Read the GOAL and REFERENCES
   - Explore the codebase to understand what exists
   - Decide scope: small (<1000 lines changed), medium (2-5 PRs), or large (needs design doc)

2. **COMMENT your plan** on this issue before starting:
   - "Scope: small/medium/large"
   - For small: "Will implement directly and open PR"
   - For medium: "Breaking into N sub-issues: [list]"
   - For large: "Will draft design doc first as PR for review"

3. **EXECUTE** based on scope:

   **SMALL** (direct PR):
   - Create branch from main
   - Implement the change
   - Run `make check` — fix any failures
   - Open PR linking this issue
   - Request review from another agent or root
   - Address review feedback
   - Merge after approval + CI green

   **MEDIUM** (sub-issues):
   - Create sub-issues linking back to this one
   - Each sub-issue should be small scope
   - Implement sub-issues in dependency order
   - Each gets its own PR with review
   - Comment on this issue as sub-issues complete
   - Close this issue when all sub-issues are done

   **LARGE** (design first):
   - Research: read codebase, search web if needed
   - Write design doc in the relevant `docs/` subdirectory (e.g. `docs/backend/`, `docs/frontend/`, `docs/infrastructure/`)
   - Open PR with just the design doc for review
   - After design approval, create phased sub-issues
   - Each phase follows medium/small workflow
   - Comment on this issue with progress updates

4. **THROUGHOUT:**
   - Comment on this issue with decisions, blockers, and findings
   - If you discover the issue is already fixed, close it with evidence
   - If you discover overlap with another issue, comment noting it
   - If scope grows beyond original GOAL, comment and ask for guidance
   - Never merge without review approval and green CI

</details>
