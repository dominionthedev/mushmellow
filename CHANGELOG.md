# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
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
- `state.NewRunID` collided on runs invoked in quick succession — it
  took the *leading* hex digits of a nanosecond timestamp, which are
  the digits least likely to differ between nearby calls. Replaced
  with `crypto/rand`. Regression test added
  (`TestNewRunID_NoCollisions`).
- README's `go install ...@latest` instruction was unusable — the
  module has no tags or public releases yet. `Makefile`'s `.PHONY`
  was also missing the `install` target.

### Known limitations
- `evaluateReadiness` reports two different situations as the same
  `blocked` status: "an upstream actually failed" and "a condition
  (e.g. `on:"failure"` when upstream succeeded) can never be
  satisfied." A real `skipped` status is a genuine gap, not hidden —
  see code comment in `internal/scheduler/scheduler.go`.
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
