# bc v2 Vision - Clarifications

## Root Agent

**Q1. Root crash recovery - what's the "break glass" procedure for humans to bypass?**
A: Not sure can you clarify more on this.

**Q2. Root bottleneck - with many engineers, could root become a merge bottleneck? Auto-merge for green CI?**
A: The root should get a few branches to merge from manager, in large teams there will be a tree like structure so root should not become the bottleneck as each of it's reportees will merge work from there reportees who will merge from there reportees and hence work can go on without problems.

**Q3. Who spawns root? `bc init` creates it automatically, or `bc up`?**
A: bc init should create root and nothing else, bc up starts the root agent.

**Q4. What if `.bc/roles/root.md` is missing? Error on init or auto-generate default?**
A: auto-generate default.

---

## Dual Queue System

**Q5. Queue persistence: Per-agent in separate files, or unified?**
A: Queues are per agent, in the agents directory (separate files).

**Q6. Epic branch lifecycle: Where does manager create it? How do engineer task branches relate?**
A: Manager creates it in its own worktree, merges all tasks from different engineers, builds the epic branch there.

**Q7. Partial completion: Can manager merge task-1 while task-2 is still in progress?**
A: Yes, manager reviews and merges tasks as they come in. Epic marked complete only when all tasks are merged.

**Q8. Queue item identity: Is it branch name? What if same branch resubmitted after rejection?**
A: Queue items are beads tasks (bd issues) with a branch associated to them.

**Q9. Conflict detection: At submit time or at merge time?**
A: Merge time

**Q10. Can engineer resubmit directly to root after fixing, or must go through manager again?**
A: Must go through manager. Basically manager gets a reject, tries to rebase, finds which branch causes conflicts and asks the engineer to fix that branch and reopens the issue associated, gets another merge request, merges, and sends back to root.

**Q11. Stale branches: If engineer's branch is behind, who triggers rebase? Auto-detect on submit?**
A: Auto-detect on submit.

**Q12. Multi-level hierarchy: Can we have Root → Manager → Sub-Manager → Engineer? Or max 2 levels?**
A: We can have multi-level tree-like org structure.

---

## Agent Memory

**Q13. Memory ownership: Tied to agent instance or role? Does new engineer-01 inherit old engineer-01's memories?**
A: Memory is tied to agent instance. Roles should have a simple prompt on initialisation and setup memory when they start.

**Q14. Memory isolation: Per-agent only, or can agents access each other's memories?**
A: Per-agent only. Memory should not be accessible by other agents.

**Q15. Memory injection: How does memory reach the agent on spawn? Prepended to prompt? Separate context?**
A: No injection. The prompt should have the details on how to load memory when spawned.

**Q16. Recording trigger: Automatic on task complete, or agent calls `bc memory record`?**
A: Agent can call `bc agent manager-furiosa memory record` (something like this). Should be updated on each task complete.

**Q17. Memory cleanup: How is stale/incorrect memory pruned? Can agents forget?**
A: Yes, agents should be able to edit their memory on their own.

**Q18. Backend fallback: What if mem0 is unavailable? Spawn with empty memory or fail?**
A: Spawn with empty memory and a warning.

**Q19. Cross-agent memory: Team-level shared learnings? Workspace-wide project context?**
A: No cross-agent memory contamination. After a sprint or if there is some major thing happening, everyone should be able to update their own memory independently.

---

## Processes

**Q20. Process lifecycle: When owner agent stops, what happens to its processes?**
A: Process should stop and be cleared if agent stops. Also we should have graceful termination to ensure if agent goes down or something happens, things are handled properly.

**Q21. Port conflicts: What if two agents try same port?**
A: For one it will work, the other will get an error.

**Q22. Scope: Managing arbitrary shell processes, or only agent-spawned ones?**
A: Arbitrary. Even I should be able to run a process using the bc CLI, agents can use that to do the same.

---

## Teams

**Q23. Team merge queue: Does team have its own queue, or just manager queues?**
A: Just manager queues as they are responsible for the team.

**Q24. Can an agent be in multiple teams?**
A: No.

**Q25. Team dissolution: What happens to agents when team is deleted?**
A: Teams are separate from agents and is just a grouping. Agents should remain with no team or move to parent team.

---

## Migration & Compatibility

**Q26. v1 migration: Clean break or migration path for existing workspaces?**
A: Clean break. Don't worry about migration.

**Q27. Beads dependency: Required or optional?**
A: Required. It will be used by us to track instead of the current work system.

**Q28. Cost controls: Where does cost awareness fit in v2?**
A: I was not thinking about cost but we should design and have a cost dashboard as well showing the cost of each agent, or team, or workspace etc.

**Q29. Worktrees: Still using git worktrees for agent isolation like v1?**
A: Yes.

**Q30. Human override: Beyond root, can humans directly intervene? Emergency stop, manual merge?**
A: Yes, humans can do anything, but they will primarily interact with agents through direct messages or channels, or through root.

---

## Implementation Phases

**Q31. Phase ordering: PM and Manager both suggest getting merge flow right before memory. Agree?**
A: I want all the things implemented. I will let you decide on the ordering and management of this work. Do it in as many phases as you want but at the end all the features should be implemented and tested.
