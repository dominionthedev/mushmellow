# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- `Skipped` — a genuine sixth status, distinct from `Blocked`.
  `Blocked` now means an actual failure occurred somewhere in the
  chain; `Skipped` means nothing failed, the puff just never ran (an
  `on:"failure"` edge whose watched puff succeeded, a `when:`
  precondition that wasn't met, or a cascade from an upstream puff
  that was itself `Skipped`). Closes the conflation flagged in the
  previous entry.
- `state.CascadeStatus(upstream)` decides what an `on:"success"`
  dependent inherits when upstream didn't satisfy success:
  `Failed`/`Blocked` upstream → `Blocked` (a real failure happened);
  `Cancelled` upstream → `Cancelled`; anything else → `Skipped`.
- `on:"failure"` edges now also fire on a `Recovered` upstream, not
  only `Failed` — a puff that failed and was then explicitly
  recovered by its own self-handler still failed first, and an
  external `on:"failure"` watcher exists to react to failures even
  ones handled internally too.
- `PuffState.BlockedBy` renamed to `Reason` (JSON: `blocked_by` →
  `reason`) — it now explains `Blocked`, `Skipped`, and `Cancelled`
  alike, not just `Blocked`.
- `when:` external preconditions: `file_contains`, `env_set`,
  `env_equals`, `command_ok`, `port_open`. Evaluated exactly once, at
  the moment a puff's dependency edges are already satisfied and it
  would otherwise dispatch — not polled repeatedly, not pre-checked at
  parse time, since the world (a file's contents, an env var, a port)
  can change between parse and dispatch. A puff whose criteria aren't
  met ends `blocked`, with a human-readable reason
  (`when: env "X" is set`), not a silent no-op. `Criterion.Validate()`
  catches missing required fields per kind (e.g. `port_open` without a
  `port`) at parse time, not dispatch time.
- `internal/state`: Runtime State types (`Run`, `PuffState`, `Attempt`)
  and persistence to `.mushmellow/runs/<run_id>/state.json`. Five
  terminal statuses (`success`/`failed`/`recovered`/`blocked`/`cancelled`)
  plus full attempt history per puff, not just the last attempt.
- `internal/scheduler`: the Dispatcher. Executes a puff's dependency
  closure — never "the whole workflow", since that object doesn't
  exist at runtime — with per-edge readiness evaluation
  (`on:"success"`/`"failure"`/`"always"`), per-profile worker pools,
  puff-level retries with backoff, self-handlers (never swallow a
  failure unless `recover: true` is explicit), external `on:"failure"`
  dependents, and blocking member-call delegation with its own nested
  Runtime State file (referenced, never inlined).
- `on_failure: halt` implemented: on an unrecovered failure, stops
  dispatching new non-exempt puffs and sends SIGTERM (then SIGKILL
  after a grace period) to in-flight non-exempt processes. Puffs whose
  readiness depends on `on:"failure"`/`on:"always"` are exempt and
  allowed to finish.
- `mushmellow run <puff>` CLI command.
- `examples/portable`: a workspace using only shell builtins
  (`true`/`false`/`echo`), so `run` behavior can be exercised without
  cargo/go being installed — `examples/basic` stays illustrative-only,
  it's never actually executed.

### Fixed
- `evaluateReadiness` used to short-circuit to `ready=true` for any
  puff with zero `depends_on` edges, which meant a dependency-free
  puff with a `when:` block would skip the check entirely and always
  run. Caught while wiring `when:` in; regression test
  (`TestRun_When_NoDependencies_StillChecked`) added.
- `state.NewRunID` collided on runs invoked in quick succession — it
  took the *leading* hex digits of a nanosecond timestamp, which are
  the digits least likely to differ between nearby calls. Replaced
  with `crypto/rand`. Regression test added
  (`TestNewRunID_NoCollisions`).
- README's `go install ...@latest` instruction was unusable — the
  module has no tags or public releases yet. `Makefile`'s `.PHONY`
  was also missing the `install` target.

### Known limitations
- `when:` criteria evaluation happens on the main dispatch loop's scan
  (not inside a per-node goroutine), so a slow criterion (e.g.
  `port_open` against an unreachable host, up to its 2s timeout) delays
  the scan reaching other nodes later in that same pass. It does not
  hold the dispatcher's mutex while doing so, so it doesn't stall other
  puffs' status updates — just adds latency to when they're noticed as
  ready.
- Halt's SIGTERM-then-SIGKILL only covers `StepShell` processes
  currently running via `exec.Cmd`. A halt triggered while a
  `StepMember` call is mid-flight does not yet propagate the kill
  signal into the nested Dispatcher.
- Two independent branch sources converging on the same downstream
  puff (true diamond convergence across two branch sources) is
  undefined behavior. Mixed branched + unbranched dependencies on one
  node are handled correctly.
- No `when:` external preconditions yet (file content, env, command
  exit, port reachability).
- On-demand (CLI-driven, ephemeral) branching is not wired to a flag
  yet, though the underlying `ExpandPuff` function is shared and ready
  for it.
