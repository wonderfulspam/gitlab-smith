# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GitLabSmith is a GitLab CI/CD configuration refactoring and validation tool written in Go. See `implementation-state.json` for current implementation status.

## Technology Stack

- **Language**: Go (planned)
- **CLI Framework**: Cobra
- **Container**: Docker for GitLab deployment
- **Testing**: Go testing package + Testify (planned)

## Commands

### Development Commands
```bash
# Build
go build -o gitlab-smith ./cmd/gitlab-smith

# Run tests
go test ./...
go test -v ./pkg/parser  # Run specific package tests

# Lint
golangci-lint run

# Format
go fmt ./...
go vet ./...

# Install dependencies
go mod download
go mod tidy
```

### GitLabSmith CLI Usage
```bash
# Parse and display GitLab CI configuration
gitlab-smith parse .gitlab-ci.yml

# Static analysis mode
gitlab-smith refactor --old .gitlab-ci.yml --new .gitlab-ci-new.yml

# With table output format
gitlab-smith refactor --old .gitlab-ci.yml --new .gitlab-ci-new.yml --format table

# Full testing mode with local GitLab
gitlab-smith refactor --old .gitlab-ci.yml --new .gitlab-ci-new.yml --full-test
```

## Architecture

### Project Structure
```
gitlab-smith/
├── cmd/gitlab-smith/       # CLI entry point
├── pkg/
│   ├── parser/            # GitLab CI YAML parser & dependency mapping
│   ├── differ/            # Semantic diffing engine
│   ├── analyzer/          # Static analysis (rules, optimization)
│   ├── deployer/          # GitLab/runner deployment management
│   └── renderer/          # Pipeline rendering & comparison
├── internal/
│   ├── config/            # Configuration management
│   └── gitlab/            # GitLab API client wrapper
├── test/
│   └── fixtures/          # Test GitLab CI configurations
├── implementation-state.json  # Current implementation status
└── pipeline-emulator-spec.md  # Detailed specification
```

### Core Components

1. **Parser Module** (`pkg/parser/`): Parses GitLab CI YAML and builds dependency graphs
2. **Differ Module** (`pkg/differ/`): Performs semantic comparison between configurations
3. **Analyzer Module** (`pkg/analyzer/`): Static analysis for common issues and optimizations
4. **Deployer Module** (`pkg/deployer/`): Manages local GitLab instance deployment
5. **Renderer Module** (`pkg/renderer/`): Renders and compares pipeline executions

### Implementation Phases

**Phase 1**: Core Analysis & Semantic Diffing
- GitLab CI parser with dependency mapping
- Semantic differ for configuration comparison
- Static analyzer with rule engine
- Basic CLI with static analysis mode

**Phase 2**: Local GitLab Deployment
- Docker-based GitLab deployment
- Pipeline rendering comparison
- Visual diff output

**Phase 3**: Performance Testing
- GitLab runner deployment
- Pipeline execution and benchmarking
- Performance comparison reports

## Key Implementation Notes

- The tool operates in two modes: static analysis (using GitLab API) and full testing (local GitLab deployment)
- Static mode provides quick feedback without infrastructure requirements
- Full testing mode validates actual pipeline behavior changes
- Focus on semantic differences, not syntactic changes
- Performance testing requires actual runner deployment and job execution

## Test Infrastructure

### Test Organization
```
test/
├── fixtures/                      # Simple test files for basic parsing
│   ├── simple.gitlab-ci.yml
│   └── simple-modified.gitlab-ci.yml
├── simple-refactoring-cases/      # Paired before/after YML files for simple tests
│   ├── *-before.yml              # Original configuration
│   └── *-after.yml               # Refactored version
├── refactoring-scenarios/         # Complex multi-file test scenarios
│   └── scenario-N/               # Each scenario is a directory
│       ├── before/               # Original configuration directory
│       │   └── .gitlab-ci.yml
│       ├── after/                # Refactored configuration directory
│       │   ├── .gitlab-ci.yml
│       │   └── ci/              # Include files (if using includes)
│       ├── includes/             # Shared includes (legacy, being migrated to after/ci/)
│       └── config.yaml           # Test expectations and metadata
└── realistic-app-scenarios/       # Real-world application examples
    └── flask-microservice/       # Complete Flask app with CI/CD
```

### Key Test Patterns
- **Simple refactoring cases**: Use paired files (`*-before.yml` and `*-after.yml`)
- **Complex scenarios**: Use directories with `before/` and `after/` subdirectories
- **Include files**: Should be in `after/ci/` or `before/ci/` relative to the main .gitlab-ci.yml
- **Test expectations**: Defined in `config.yaml` for each scenario

### Running Tests
```bash
# All tests
go test ./...

# Specific scenario
go test ./pkg/validator -run TestRefactoringScenarios/scenario-6 -v

# Simple refactoring tests
go test ./pkg/validator -run TestSimpleRefactoringCases -v
```

## Getting Started

To begin implementation:
1. Initialize Go module: `go mod init github.com/yourusername/gitlab-smith`
2. Set up Cobra CLI structure in `cmd/gitlab-smith/`
3. Implement Phase 1 starting with the parser module
4. Use the specification in `pipeline-emulator-spec.md` as the detailed guide

**Current Status**: See `implementation-state.json` for detailed implementation progress.