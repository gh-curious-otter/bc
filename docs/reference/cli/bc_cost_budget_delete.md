## bc cost budget delete

Delete a budget

### Synopsis

Delete a budget configuration.

Examples:
  bc cost budget delete                  # Delete workspace budget
  bc cost budget delete --agent eng-01   # Delete agent budget

```
bc cost budget delete [flags]
```

### Options

```
      --agent string   Delete budget for specific agent
  -h, --help           help for delete
      --team string    Delete budget for specific team
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc cost budget](bc_cost_budget.md)	 - Manage cost budgets

