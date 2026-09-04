# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- `internal/puff`: core domain types — `Puff`, `Profile`, `Step`,
  `DependsOn`, `Criterion`, `ArtifactDecl`, `OnFailure`, `BranchSpec`.
- `internal/workspace`: parsing of `mushmellow.yaml` / `*.mushmellow.yaml`,
  workspace-root discovery, member discovery, override-by-key context merge.
- `internal/graph`: DAG construction from parsed puffs, unknown-dependency
  validation, cycle detection, static (matrix) branch expansion with
  edge-local downstream cascade, shared `ExpandPuff` mechanism intended
  for future on-demand branching too.
- `mushmellow validate` CLI command: end-to-end pipeline check (parse →
  discover members → build graph) with a readable report.
- Example workspace (`examples/basic`) demonstrating a member call, a
  branch matrix, a self-handler, and an external `on:"failure"` dependent.

### Known limitations
- Two independent branch sources converging on the same downstream puff
  (true diamond convergence across two branch sources) is undefined
  behavior. Mixed branched + unbranched dependencies on one node are
  handled correctly.
- No Scheduler/Dispatch yet — puffs don't execute. The graph is built and
  validated but nothing runs.
- No Runtime State persistence yet.
- On-demand (CLI-driven, ephemeral) branching is not wired to a flag yet,
  though the underlying `ExpandPuff` function is shared and ready for it.
