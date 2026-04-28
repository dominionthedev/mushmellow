# 🍡 Mushmellow

**Soft workflows. Hard execution.**

Mushmellow is a lightweight, stylish developer workflow runtime. It allows you to define and execute structured developer flows called **mushmellows**, composed of dependency-aware units called **puffs**.

## Why Mushmellow?

- **Readable**: YAML-based configuration that anyone can understand.
- **Portable**: Run the same workflows locally and in CI.
- **Aesthetic**: Styled output using Lipgloss for a "soft" developer experience.
- **Smart**: Dependency-aware execution with a Directed Acyclic Graph (DAG).
- **Local-first**: Designed for developers, by developers.

## Installation

```bash
go build -o /usr/local/bin/mushmellow .
```

## Core Concepts

### 🍡 Mushmellow
A named workflow (e.g., `build`, `test`, `release`).

### ☁️ Puff
The atomic execution unit. A puff can run commands, display messages, or wait. Puffs can depend on other puffs, forming an execution graph.

## Quick Start

Create a `mushmellow.yaml` in your project root:

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
- `doctor`: Validate your configuration.
- `edit`: Open configuration in your default editor.

## License

MIT
