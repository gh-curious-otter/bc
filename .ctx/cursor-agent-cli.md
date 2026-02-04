# Using Cursor Agent CLI with bc

To have bc workers behave like Claude Code (all permissions, no confirmations, and able to receive input via `bc send` + Enter), run Cursor Agent with these flags.

## Recommended command

```bash
cursor-agent --force --print
```

- **`--force`** — All permissions: allow all commands unless explicitly denied. No confirmation prompts for tool use (like Claude Code’s “skip permissions”).
- **`--print`** — Headless/scriptable mode: prints responses to the terminal, has access to all tools (write, bash, etc.). Stdin is used for input, so when bc sends a message and then Enter, the agent receives and runs it.

## In bc

The default agent command in `config.toml` is set to:

```toml
[agent]
command = "cursor-agent --force --print"
```

So `bc up` starts workers with the above. If you override via workspace `.bc/config.json` or `bc up --agent cursor-agent`, the `[[agents]]` entry for `cursor-agent` also uses `cursor-agent --force --print`.

## Other useful flags

- **`--approve-mcps`** — Auto-approve MCP servers (only with `--print`).
- **`--workspace <path>`** — Set workspace (bc sets `BC_WORKSPACE` and runs the agent in that directory).
- **`--model <model>`** — e.g. `sonnet-4`, `gpt-5`.

## Existing workspaces

If you already have a bc workspace, its `.bc/config.json` may have `agent_command` set to `cursor-agent` without flags. Either:

- Edit `.bc/config.json` and set `"agent_command": "cursor-agent --force --print"`, or  
- Remove the `agent_command` key so bc uses the default from `config.toml`.

Then run `bc down` and `bc up` so workers restart with the new command.

## If you use `cursor` (GUI) subcommand

The GUI Cursor uses a different command. For headless workers with bc, use **cursor-agent** (the CLI binary) with **--force --print**.

## Reference

From `cursor-agent --help`:

- `-f, --force` — Force allow commands unless explicitly denied.
- `-p, --print` — Print responses to console (for scripts or non-interactive use). Has access to all tools, including write and bash.
