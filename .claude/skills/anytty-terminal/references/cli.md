# AnyTTY CLI reference

## Preflight

```sh
anytty --version
anytty endpoint list --json
anytty endpoint show build-server --json
anytty endpoint test build-server --json
anytty terminal list --endpoint build-server --json
```

`endpoint list` reads the configured registry without dialing endpoints. `endpoint test` verifies protocol reachability. Check `anytty daemon status` only when operating the Local endpoint.

If the user selected an endpoint, pass `--endpoint ENDPOINT` when listing or creating terminals. Without it, AnyTTY uses the registry default. A stable target has the form `ENDPOINT:TERMINAL`, and commands that accept a target route through its owning endpoint.

Use `terminal list --all-endpoints --json` only when the task needs cross-endpoint discovery because it connects to every enabled endpoint.

## Operate an existing remote terminal

```sh
anytty terminal list --endpoint build-server --json
anytty terminal show build-server:release --json
anytty terminal capture build-server:release --live --json
anytty terminal send build-server:release --literal "continue" --enter --json
anytty terminal capture build-server:release --lines 300 --json
```

Inspect the latest Live screen immediately before sending input. Literal input is written to the PTY like keyboard input; it is not a remote shell command API. Use `--key Ctrl-C`, `--key Enter`, or another named key for control input.

## Run a command on a remote endpoint

Prefer direct argv when the executable and arguments are known:

```sh
anytty terminal create --endpoint build-server --json --name check-repo \
  --tag kind=task --tag agent=codex --tag task=status \
  --cwd /srv/project -- git status --short

anytty terminal wait build-server:check-repo --state exited --timeout 5m --json
anytty terminal show build-server:check-repo --json
anytty terminal capture build-server:check-repo --lines 300 --json
```

Use a shell only for pipes, redirection, expansion, or compound syntax. Match the endpoint host rather than assuming a POSIX shell:

```sh
anytty terminal create --endpoint build-server --json --name test-and-log \
  --tag kind=task --tag agent=codex --tag task=test \
  --cwd /srv/project -- sh -lc 'go test ./... 2>&1 | tee test.log'
```

The create result is a finite JSON envelope. Retain its returned target instead of constructing one from the requested name:

```json
{"schema_version":1,"kind":"terminal_created","target":"build-server:check-repo","state":"running"}
```

## Inspect and transfer endpoint files

Read-only operations:

```sh
anytty file list build-server /srv/project --all --json
anytty file stat build-server /srv/project/test.log --json
anytty file cat build-server /srv/project/test.log
anytty file download build-server /srv/project/test.log ./test.log --json
```

Transfers and mutations:

```sh
anytty file upload build-server ./config.new /srv/project/config.new --json
anytty file mkdir build-server /srv/project/output --parents --json
anytty file rename build-server /srv/project/config.new /srv/project/config --json
anytty file copy build-server /srv/project/a /srv/project/b /srv/project/output --json
anytty file move build-server /srv/project/old /srv/project/archive --json
anytty file remove build-server /srv/project/stale --json
```

Do not add `--overwrite`, `--recursive`, or a destructive mutation unless it is within the user's requested scope. `file cat` writes raw file content to stdout; use `download` when preserving the file locally matters.

## Create a task terminal

Use a shared `run` tag to group terminals created for one parent job:

```sh
anytty terminal create --endpoint build-server --json --name fix-login-tests \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=tests \
  --cwd /path/to/project -- go test ./...
```

The terminal name is also its daemon-local ID at creation, so inspect the pool first and choose a unique name. Keep every argument after `--` separate so AnyTTY records the exact process specification.

The create result is a finite JSON envelope:

```json
{"schema_version":1,"kind":"terminal_created","target":"build-server:fix-login-tests","state":"running"}
```

Use the returned `target` for later operations.

## Run tasks in parallel

`terminal create` returns after the process is started, so independent tasks can be created one after another and continue concurrently:

```sh
anytty terminal create --endpoint build-server --json --name fix-login-tests \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=tests \
  --cwd /path/to/project -- go test ./...

anytty terminal create --endpoint build-server --json --name fix-login-build \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=build \
  --cwd /path/to/project -- npm run build

anytty terminal list --endpoint build-server --tag kind=task --tag run=fix-login --json
```

Use a shell only when the task actually needs pipes, redirection, expansion, or compound shell syntax:

```sh
anytty terminal create --endpoint build-server --json --name fix-login-build-log \
  --tag kind=task --tag agent=codex --tag run=fix-login --tag task=build \
  --cwd /path/to/project -- sh -lc 'npm run build 2>&1 | tee build.log'
```

## Inspect and monitor a task

```sh
anytty terminal show build-server:fix-login-tests --json
anytty terminal capture build-server:fix-login-tests --live --json
anytty terminal events build-server:fix-login-tests --output ndjson --timeout 30s
anytty terminal wait build-server:fix-login-tests --state exited --timeout 30m --json
anytty terminal show build-server:fix-login-tests --json
anytty terminal capture build-server:fix-login-tests --lines 200 --json
```

`terminal show` and list items expose stable lowercase fields including:

- `target`, `endpoint_id`, `terminal_id`, and `name`
- `command` as an argv array and `cwd`
- `tags`, `state`, `cols`, and `rows`
- `created_at`, `last_output_at`, `exit_code`, and `exited_at`
- `foreground_process`, when available, as a normalized name without argv

`terminal capture --live` returns the latest native screen. Capture without `--live` projects authoritative history and is suitable for replay or summary. Event output is NDJSON, one envelope per line, and can be filtered with repeated `--type` flags.

## Take over or answer a prompt

```sh
anytty terminal attach build-server:fix-login-tests
anytty terminal send build-server:fix-login-tests --literal "continue" --enter --json
anytty terminal send build-server:fix-login-tests --key Ctrl-C --json
```

Literal input is not a shell command API; it is written to the PTY exactly like keyboard input. Always inspect the screen first. Named keys include values such as `Enter`, `Ctrl-C`, `Up`, and `Escape`.

For exact future PTY bytes, use:

```sh
anytty terminal stream build-server:fix-login-tests
```

`stream` does not replay old output. Use capture for existing history.

## History and lifecycle

```sh
anytty terminal capture build-server:fix-login-tests --lines 1000 --json
anytty terminal restart build-server:fix-login-tests --json
anytty terminal kill build-server:fix-login-tests --json
anytty terminal remove build-server:fix-login-tests --json
```

`history search` is local-daemon-specific and accepts a terminal ID rather than a remote target. For Local, SSH, Direct, or Cloud endpoints generally, use `terminal capture TARGET` so the owning daemon supplies authoritative history.

Restart reuses the stored process specification. Kill preserves the terminal record and history. Remove is valid only after exit and should be treated as cleanup, not as the way to stop a task.

The task terminal itself is the unit of observation. AnyTTY does not need to inspect the task's internal process tree to report its name, command, state, exit code, output, and history.
