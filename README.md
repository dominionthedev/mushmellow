<p align="center">
  <a href="https://github.com/dominionthedev/mushmellow">
    <img src="assets/logo.svg" alt="MushMellow Logo" width="800"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/dominionthedev/mushmellow/releases">
    <img src="https://img.shields.io/github/v/release/dominionthedev/mushmellow?style=for-the-badge&color=A855F7" alt="Release"/>
  </a>
  <a href="https//github.com/dominionthedev/mushmellow/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/dominionthedev/mushmellow/test.yaml?style=for-the-badge&color=22D3EE" alt="Build Status"/>
  </a>
  <a href="./LICENSE">
    <img src="https://img.shields.io/github/license/dominionthedev/mushmellow?style=for-the-badge&color=FF4FD8" alt="License"/>
  </a>
</p>

---

**Soft workflows. Hard execution.**

Mushmellow is a lightweight, stylish developer workflow runtime. It allows you to define and execute structured developer flows called **mushmellows**, composed of dependency-aware units called **puffs**.

## Why Mushmellow?

- **Readable**: YAML-based configuration that anyone can understand.
- **Portable**: Run the same workflows locally and in CI.
- **Composable**: Supports multi-file configurations via `*.mushmellow.yaml` discovery.
- **Aesthetic**: Styled output using Lipgloss for a "soft" developer experience. ✨
- **Smart**: Dependency-aware execution with a Directed Acyclic Graph (DAG).
- **Local-first**: Designed for developers, by developers.

## Installation

### From Source
```bash
go build -o /usr/local/bin/mushmellow .
```

### Via Go
```bash
go install github.com/dominionthedev/mushmellow@latest
```

## Core Concepts

### 🍡 Mushmellow
A named workflow (e.g., `build`, `test`, `release`).

### ☁️ Puff
The atomic execution unit. A puff can run commands, display messages, or wait. Puffs can depend on other puffs, forming an execution graph.

## Quick Start

Mushmellow automatically discovers `mushmellow.yaml` or any `*.mushmellow.yaml` in your current directory.

Create a `dev.mushmellow.yaml`:

```yaml
version: 1
name: My Project

mushmellows:
  dev:
    description: My dev workflow
    puffs:
      - id: hello
        type: message
        text: "Starting dev flow..."
      - id: check
        run: "go version"
      - id: build
        depends_on: [check]
        run: "go build ."
```

Run it:

```bash
mushmellow run dev
```

## CLI Commands

- `run <name>`: Execute a workflow.
- `list`: List all available workflows.
- `new <name>`: Scaffold a new workflow file.
- `puff`: Manage puffs via CLI.
- `doctor`: Validate your configuration.
- `edit`: Open configuration in your default editor.

## License

This project is licensed under the MIT License.

---

<br/>

<p align="center">
DominionDev
<a href="https://github.com/dominionthedev">GitHub</a> • <a href="https://dominionthedev.github.io">Website</a>
</p>

<p align="center">
  <a href="https://dominionthedev.github.io">
    <img src="https://raw.githubusercontent.com/dominionthedev/dominionthedev/main/assets/watermark.svg" alt="DominionDev" width="1024"/>
  </a>
</p>
