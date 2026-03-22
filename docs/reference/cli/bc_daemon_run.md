## bc daemon run

Run a named workspace process

### Synopsis

Run a long-lived workspace process in a tmux session or Docker container.

Examples:
  bc daemon run --name api --runtime tmux --cmd "go run ./cmd/api"
  bc daemon run --name db --runtime docker --image postgres:17 --port 5432:5432

```
bc daemon run --name <name> --runtime <tmux|docker> [options] [flags]
```

### Options

```
      --cmd string           Command to run (tmux runtime)
  -d, --detach               Run in background (default true) (default true)
      --env stringArray      Env var KEY=VALUE (repeatable)
      --env-file string      File of KEY=VALUE env vars
  -h, --help                 help for run
      --image string         Docker image (docker runtime)
      --name string          Process name (required)
      --port stringArray     Port mapping, e.g. 5432:5432 (repeatable)
      --restart string       Restart policy: no|always|on-failure (default "no")
      --runtime string       Runtime: tmux or docker (required)
      --volume stringArray   Volume mount, e.g. /var/run/docker.sock:/var/run/docker.sock (repeatable)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc daemon](bc_daemon.md)	 - Manage workspace processes and the bcd server

