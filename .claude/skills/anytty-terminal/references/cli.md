# AnyTTY task terminal reference

## Preflight

```sh
anytty daemon status
anytty endpoint list
anytty terminal list --all-endpoints --json
```

If the user selected an endpoint, pass `--endpoint ENDPOINT` consistently. Without it, AnyTTY uses the registry default. A stable target has the form `ENDPOINT:TERMINAL`.

## Create a task terminal

Use a shared `run` tag to group terminals created for one parent job:

```sh
anytty terminal create --json --name fix-login-tests \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=tests \
  --cwd /path/to/project -- go test ./...
```

The terminal name is also its daemon-local ID at creation, so inspect the pool first and choose a unique name. Keep every argument after `--` separate so AnyTTY records the exact process specification.

The create result is a finite JSON envelope:

```json
{"schema_version":1,"kind":"terminal_created","target":"local:fix-login-tests","state":"running"}
```

Use the returned `target` for later operations.

## Run tasks in parallel

`terminal create` returns after the process is started, so independent tasks can be created one after another and continue concurrently:

```sh
anytty terminal create --json --name fix-login-tests \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=tests \
  --cwd /path/to/project -- go test ./...

anytty terminal create --json --name fix-login-build \
  --tag kind=task --tag agent=codex --tag project=shop \
  --tag run=fix-login --tag task=build \
  --cwd /path/to/project -- npm run build

anytty terminal list --tag kind=task --tag run=fix-login --json
```

Use a shell only when the task actually needs pipes, redirection, expansion, or compound shell syntax:

```sh
anytty terminal create --json --name fix-login-build-log \
  --tag kind=task --tag agent=codex --tag run=fix-login --tag task=build \
  --cwd /path/to/project -- sh -lc 'npm run build 2>&1 | tee build.log'
```

## Inspect and monitor a task

```sh
anytty terminal show local:fix-login-tests --json
anytty terminal capture local:fix-login-tests --live --json
anytty terminal events local:fix-login-tests --output ndjson --timeout 30s
anytty terminal wait local:fix-login-tests --state exited --timeout 30m --json
anytty terminal show local:fix-login-tests --json
anytty terminal capture local:fix-login-tests --lines 200 --json
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
anytty terminal attach local:fix-login-tests
anytty terminal send local:fix-login-tests --literal "continue" --enter --json
anytty terminal send local:fix-login-tests --key Ctrl-C --json
```

Literal input is not a shell command API; it is written to the PTY exactly like keyboard input. Always inspect the screen first. Named keys include values such as `Enter`, `Ctrl-C`, `Up`, and `Escape`.

For exact future PTY bytes, use:

```sh
anytty terminal stream local:fix-login-tests
```

`stream` does not replay old output. Use capture for existing history.

## History and lifecycle

```sh
anytty history search fix-login-tests "error" --fixed-strings --context 2
anytty terminal restart local:fix-login-tests --json
anytty terminal kill local:fix-login-tests --json
anytty terminal remove local:fix-login-tests --json
```

`history search` searches history owned by the local daemon and accepts the terminal ID, not a remote target. For Local, SSH, Direct, or Cloud endpoints generally, use `terminal capture TARGET` so the owning daemon supplies authoritative history.

Restart reuses the stored process specification. Kill preserves the terminal record and history. Remove is valid only after exit and should be treated as cleanup, not as the way to stop a task.

The task terminal itself is the unit of observation. AnyTTY does not need to inspect the task's internal process tree to report its name, command, state, exit code, output, and history.
