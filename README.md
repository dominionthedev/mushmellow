# 🍡 mushmellow

**Soft workflows. Hard execution.**

A deterministic, local-first orchestration runtime for development
workflows. You declare puffs (units of work) and their dependencies in
YAML; mushmellow resolves them into a DAG, runs what's runnable
concurrently within resource budgets you define, and gives you an honest
account of what happened and why.

> **Status: pre-alpha.** The dependency graph, YAML parsing, and branch
> expansion are implemented and tested. Nothing executes yet — there is
> no scheduler or runner. See [Roadmap](#roadmap) and
> [CHANGELOG.md](./CHANGELOG.md) for exactly what's real today.

## Why

Most task runners (Make, Taskfile) give you dependency ordering and
nothing else. Most workflow engines (GitHub Actions, Dagger) assume
disposable cloud runners and bake in their own isolation model.
Mushmellow is neither: it's built for a local machine you actually care
about the resource usage of, with a failure model precise enough to
tell you not just "it failed" but *what* failed, *what got blocked
because of it*, and *whether a handler already dealt with it*.

Mushmellow does not attempt to provide isolation itself. If you want a
puff sandboxed, run mushmellow inside something that provides that
(a container, [Runbox](https://github.com/dominionthedev/runbox), etc).
That boundary is external and honest, not implied by a config field
mushmellow can't actually enforce.

## Concepts

- **Workspace** — the root `mushmellow.yaml`. Where execution is valid.
- **Puff** — the smallest unit of orchestration. A named sequence of
  steps (`run` a command, `echo` a message, or call into a `member`),
  with optional dependencies, a profile, retry/pool overrides, declared
  artifacts, a failure self-handler, and branch variance.
- **Profile** — a puff's execution environment: env vars, shell,
  working directory, and a worker-pool concurrency budget. No
  "isolation type" — see [Why](#why).
- **Branching** — declare a puff as a matrix (e.g. vary an env var
  across values) and mushmellow expands it into sibling puffs at graph-build
  time, cascading the expansion to every puff downstream of it. Fully
  static and deterministic — no runtime graph mutation.
- **Members** — a subfolder with its own `*.mushmellow.yaml` is a
  self-contained workspace a parent puff can call into (blocking,
  synchronous), without merging into one cross-folder graph.
- **Failure model** — a failed puff blocks everything downstream of it
  (reachability from the failure point), nothing else. Puffs can
  declare a self-handler (cleanup/notify, doesn't swallow the failure
  unless it explicitly recovers) or another puff can watch for a
  failure externally via `depends_on(x, on: "failure")`.

## Try it

```sh
make build
make validate
```

`make validate` parses `examples/basic/mushmellow.yaml`, discovers its
`cmd/api` member, expands the `cargo_build` branch matrix, cascades that
expansion into `test` and `notify_failure`, runs cycle detection, and
prints the fully resolved graph — proof the pipeline works end to end,
not just that it compiles.

```sh
make test    # unit tests
make vet     # go vet
make fmt     # gofmt -w .
```

## Example

```yaml
puffs:
  build:
    profile: build
    steps:
      - echo: "building api"
      - member: cmd/api::build
      - echo: "done building api"
    on_failure:
      steps:
        - echo: "failed to build API"

  cargo_build:
    profile: build
    steps:
      - run: cargo build --release
    artifacts:
      - path: target/release/mybin
    branch:
      matrix:
        CARGO_TARGET_DIR: ["branch-a", "branch-b"]

  test:
    depends_on:
      - puff: cargo_build
    steps:
      - run: cargo test
```

## Roadmap

- [ ] Scheduler / Dispatch — actually run the graph
- [ ] Runtime State persistence (`.mushmellow/runs/<id>/state.json`)
- [ ] `on_failure: isolate | halt` enforcement
- [ ] On-demand (CLI-driven, ephemeral) branching
- [ ] `when:` external preconditions (file content, env, command exit,
      port reachability)
- [ ] Lua DSL, once YAML stops being enough

## License

[MIT](./LICENSE)
