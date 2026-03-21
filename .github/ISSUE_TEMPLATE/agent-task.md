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
   - Check for dependency comments — do NOT start if a dependency issue is still open
   - Explore the codebase to understand what exists
   - Decide scope: small (<1000 lines changed), medium (2-5 PRs), or large (needs design doc)

2. **COMMENT your plan** on this issue before starting:
   - "Scope: small/medium/large"
   - For small: "Will implement directly and open PR"
   - For medium: "Breaking into N sub-issues: [list]"
   - For large: "Will write design doc in this issue for review"

3. **EXECUTE** based on scope:

   **SMALL** (direct PR):
   - Create branch from main
   - Implement the change
   - Run `make check` — fix any failures
   - Open PR linking this issue with the correct domain label (frontend/backend/infrastructure)
   - Address review feedback
   - Merge after approval + CI green

   **MEDIUM** (sub-issues):
   - Create 4-5 sub-issues linking back to this one
   - Each sub-issue should be small scope with clear dependencies noted
   - Add dependency comments between sub-issues (e.g. "Depends on #X")
   - Pick the first sub-issue (no blockers) and implement it
   - Each gets its own PR with the correct domain label
   - Comment on this issue as sub-issues complete
   - This issue stays open until all sub-issues are done

   **LARGE** (design first):
   - Research: read codebase, explore relevant packages
   - Write the design doc **as a comment on this issue** (not as a separate PR or file)
   - Include: problem statement, proposed phases, file changes per phase, migration plan
   - Wait for review/approval of the design comment
   - After approval, create medium-sized phase sub-issues (each phase is a "medium" scope)
   - Each phase gets its own sub-issues broken into small PRs
   - Comment on this issue with progress updates
   - This issue stays open until all phases are done

4. **THROUGHOUT:**
   - Always add the correct domain label to your PR (frontend/backend/infrastructure)
   - Comment on this issue with decisions, blockers, and findings
   - If you discover the issue is already fixed, close it with evidence
   - If you discover overlap with another issue, comment noting it
   - If scope grows beyond original GOAL, comment and ask for guidance
   - Never merge without review approval and green CI
   - Do NOT close parent/epic issues when merging sub-issue PRs

</details>
