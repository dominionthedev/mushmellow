# Contributing

Mushmellow is a solo-maintained project. It's open for reading, forking,
and issue reports; PRs are welcome but will be judged against the design
discipline the project is built on, not just "does it work."

## Ground rules

- **Scope discipline.** A fix PR stays a fix PR. Don't smuggle in new
  architecture, renamed concepts, or new config fields alongside a bugfix.
  If you think something bigger needs to change, open an issue and discuss
  it before writing code.
- **No invented work.** Don't add unrequested changes to look thorough.
  Smaller, focused diffs are preferred over sweeping ones.
- **Every concept needs mechanics, not just a name.** If you're proposing
  a new field or construct, it needs a concrete answer to "what happens
  when X" for the actual edge cases, not just a description of the happy
  path.
- **Tests must catch the specific bug they fix.** A regression test that
  would pass against the buggy code isn't a regression test.

## Conventional commits

This repo uses [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): short summary

optional body explaining why, not just what
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`.
Scope is usually a package name (`puff`, `graph`, `workspace`, `cmd`).

## Before opening a PR

```
make fmt
make vet
make test
make validate
```

All four should be clean. CI runs the same checks.

## Reporting issues

Include the `mushmellow.yaml` (or a minimal reproduction) that triggers
the problem. "It doesn't work" without a reproducible workspace isn't
actionable.
