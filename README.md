# agentflow-core

A Go-based agent workflow engine.

## Quick Start

```bash
# Build
make build

# Run
make run

# Test
make test

# Docker
make docker-build
make docker-run
```

## Project Structure

```
agentflow-core/
├── cmd/          # Application entry points
├── internal/     # Private application code
│   └── storage/  # Storage layer implementation
├── pkg/          # Public packages (models, interfaces)
├── api/          # API definitions and protocols
└── data/         # Data files (SQLite DB, etc.)
```
