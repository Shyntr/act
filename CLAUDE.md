# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**shyntr/act** is a fork of [nektos/act](https://github.com/nektos/act),
used as the execution engine library for Shyntr Pipe CI/CD platform.

Key changes from upstream:
- `pkg/container/k8s.go` — Kubernetes Pod executor (new, our code)
- `pkg/container/factory.go` — driver selection: "docker" or "k8s" (modified)
- `cmd/` — CLI removed, library-only use
- `go.mod` — module renamed to `github.com/shyntr/act`

Upstream: `nektos/master` branch mirrors nektos/act master exactly.
Our changes live on: `shyntr/main` branch.

## Fork Rules — Critical

NEVER modify these files (upstream owned):
- `pkg/model/` — workflow YAML parser
- `pkg/runner/` — execution engine, expression evaluator
- `pkg/common/` — Executor pattern
- `pkg/exprparser/` — ${{ }} expression interpreter
- `pkg/artifacts/` and `pkg/artifactcache/`

ONLY modify or add:
- `pkg/container/k8s.go` — K8s executor implementation
- `pkg/container/factory.go` — driver selection logic
- `go.mod` / `go.sum` — module rename only

## Upstream Sync

When nektos/act releases a new version:
```bash
git fetch upstream
git checkout nektos/master
git merge upstream/master
git push origin nektos/master

git checkout shyntr/main
git merge nektos/master
# Conflicts only expected in pkg/container/factory.go and go.mod
```

## Common Commands

- `make build` — build binary to `dist/local/act`
- `make test` — run `go test ./...`
- `make lint-go` — run `golangci-lint run`
- `make format` — run `go fmt ./...`
- `make tidy` — run `go mod tidy`
- `go test ./pkg/container/...` — test K8s executor specifically
- `go test ./pkg/runner/...` — run runner tests (upstream, should always pass)

## Architecture

### Execution Flow

1. **Planner** (`pkg/model/planner.go`) — parses workflow YAML into a `Plan`
2. **Runner** (`pkg/runner/runner.go`) — converts Plan into `Executor` chains
3. **RunContext** (`pkg/runner/run_context.go`) — holds job execution state
4. **Steps** (`pkg/runner/step.go`) — each step type implements the `step` interface
5. **Container** (`pkg/container/`) — execution environment (Docker or K8s)

### Container Driver Architecture

```
pkg/container/
├── interface.go       — ExecutionEnvironment interface
├── factory.go         — NewContainer(driver, spec) — our addition
├── docker.go          — Docker implementation (upstream)
├── k8s.go             — Kubernetes Pod implementation (ours)
└── host_environment.go — host execution (upstream)
```

The runner calls `NewContainer()` from factory — it never imports docker or k8s
directly. Switching from Docker to K8s is transparent to the runner.

### Core Abstraction: Executor Pattern

`Executor` = `func(ctx context.Context) error` — composable via:
- `.Then()`, `.Finally()`, `.OnError()` — chaining
- `NewPipelineExecutor()` — serial
- `NewParallelExecutor()` — parallel
- `.If()`, `.IfNot()` — conditional

### Key Packages

- **`pkg/model/`** — YAML parsing, plan creation (upstream, do not modify)
- **`pkg/runner/`** — core engine, expressions, step types (upstream, do not modify)
- **`pkg/container/`** — execution environments; k8s.go is our primary work area
- **`pkg/common/`** — Executor pattern, logging (upstream, do not modify)
- **`pkg/exprparser/`** — ${{ }} expression interpreter (upstream, do not modify)

## Linting Rules

- Use `errors` from stdlib, not `github.com/pkg/errors`
- Use `github.com/sirupsen/logrus` (aliased as `log`), not stdlib `log`
- Use `github.com/stretchr/testify` for tests
- Max cyclomatic complexity: 20
- Import aliases: `logrus` → `log`, `testify/assert` → `assert`

## Testing

- Tests use `testify/assert` and `testify/mock`
- Table-driven tests preferred
- Test fixtures in `testdata/` alongside packages
- K8s executor tests use `k8s.io/client-go/kubernetes/fake` — no real cluster needed
- Upstream runner tests must always pass — if they break, we broke something
