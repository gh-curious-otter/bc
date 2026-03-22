## bc cost

Show cost information

### Synopsis

Commands for viewing API cost information.

Shows Claude Code token usage, costs, and budget management.

Examples:
  bc cost                              # Show cost records (default)
  bc cost show eng-01                  # Show costs for specific agent
  bc cost usage                        # Claude Code usage via ccusage
  bc cost usage --monthly              # Monthly summary
  bc cost budget show                  # Show budget status

See Also:
  bc home           TUI dashboard with cost overview
  bc status         Agent status (includes cost info)

```
bc cost [flags]
```

### Options

```
  -h, --help   help for cost
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc cost agent](bc_cost_agent.md)	 - Show per-agent cost breakdown
* [bc cost budget](bc_cost_budget.md)	 - Manage cost budgets
* [bc cost daily](bc_cost_daily.md)	 - Show daily cost totals
* [bc cost dashboard](bc_cost_dashboard.md)	 - Show rich cost dashboard
* [bc cost model](bc_cost_model.md)	 - Show per-model cost breakdown
* [bc cost show](bc_cost_show.md)	 - Show cost records
* [bc cost summary](bc_cost_summary.md)	 - Show workspace cost overview
* [bc cost usage](bc_cost_usage.md)	 - Show Claude Code token usage via ccusage

