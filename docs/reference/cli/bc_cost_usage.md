## bc cost usage

Show Claude Code token usage via ccusage

### Synopsis

Show Claude Code token usage and cost analytics via ccusage.

Wraps the ccusage tool (https://github.com/ryoppippi/ccusage) to display
detailed token usage, per-model cost breakdown, and cache analytics from
Claude Code's local JSONL session files.

Requires npx (Node.js) to be available on the system.

Examples:
  bc cost usage                        # Daily usage report
  bc cost usage --monthly              # Monthly summary
  bc cost usage --session              # Per-session breakdown
  bc cost usage --since 20260301       # Usage since date (YYYYMMDD)
  bc cost usage --until 20260301       # Usage until date (YYYYMMDD)
  bc cost usage --json                 # Raw JSON output

```
bc cost usage [flags]
```

### Options

```
  -h, --help           help for usage
      --monthly        Show monthly summary
      --refresh        Force refresh cached data
      --session        Show per-session breakdown
      --since string   Filter from date (YYYYMMDD)
      --until string   Filter until date (YYYYMMDD)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc cost](bc_cost.md)	 - Show cost information

