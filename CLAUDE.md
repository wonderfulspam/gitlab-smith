# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GitLabSmith is a GitLab CI/CD configuration refactoring and validation tool written in Go. See `implementation-state.json` for current implementation status.

## Technology Stack

- **Language**: Go
- **CLI Framework**: Cobra
- **Container**: Docker for GitLab deployment
- **Testing**: Go testing package

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

# Required after any changes
go test ./... && go fmt ./... && go vet ./...
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

# Generate visual pipeline diagrams
gitlab-smith visualize .gitlab-ci.yml --format mermaid
gitlab-smith visualize .gitlab-ci.yml --format dot --output pipeline.dot

# Visual comparison between configurations
gitlab-smith refactor --old .gitlab-ci.yml --new .gitlab-ci-new.yml --pipeline-compare --format mermaid
gitlab-smith refactor --old .gitlab-ci.yml --new .gitlab-ci-new.yml --pipeline-compare --format dot --output comparison.dot
```

## Architecture

### Project Structure
```
gitlab-smith/
├── cmd/gitlab-smith/              # CLI entry point & commands
│   ├── main.go                   # Main CLI application
│   ├── parse.go                  # Parse command implementation
│   ├── analyze.go                # Analyze command implementation
│   ├── refactor.go              # Refactor command implementation
│   ├── visualize.go             # Visualize command implementation
│   └── *_test.go               # CLI command tests
├── pkg/
│   ├── parser/                  # GitLab CI YAML parser & workflow evaluation
│   │   ├── parser.go           # Main parsing logic
│   │   ├── types.go            # Core data structures and types
│   │   ├── resolver.go         # Include resolution logic
│   │   ├── simulation.go       # Pipeline simulation functionality
│   │   ├── workflow.go         # Workflow rules and pipeline context
│   │   └── *_test.go          # Parser tests
│   ├── analyzer/               # Static analysis with rule engine
│   │   ├── analyzer.go         # Main analyzer with 72+ rules
│   │   ├── config.go          # Configuration management
│   │   ├── registry.go        # Rule registry system
│   │   ├── types/             # Analysis result types
│   │   ├── performance/       # Performance analysis rules
│   │   ├── security/          # Security analysis rules
│   │   ├── reliability/       # Reliability analysis rules
│   │   ├── maintainability/   # Maintainability analysis rules
│   │   │   ├── maintainability.go    # Rule registration
│   │   │   ├── naming_checks.go      # Job naming validation
│   │   │   ├── complexity_checks.go  # Script and rule complexity
│   │   │   ├── duplication_checks.go # Code duplication detection
│   │   │   └── structure_checks.go   # Configuration structure
│   │   └── *_test.go         # Analyzer tests
│   ├── differ/                 # Semantic diffing engine
│   │   ├── differ.go          # Configuration comparison logic
│   │   └── *_test.go         # Differ tests
│   ├── deployer/              # GitLab deployment management
│   │   ├── deployer.go        # Docker-based GitLab deployment
│   │   └── *_test.go         # Deployer tests
│   ├── renderer/              # Pipeline rendering & visualization
│   │   ├── renderer.go        # Core renderer and GitLab API client
│   │   ├── types.go           # Pipeline execution data structures
│   │   ├── simulation.go      # Pipeline execution simulation
│   │   ├── comparison.go      # Pipeline comparison logic
│   │   ├── formatter.go       # Output formatting (table, JSON)
│   │   ├── visual.go          # Mermaid & DOT diagram generation
│   │   └── *_test.go         # Renderer tests
│   └── validator/             # Refactoring validation & test scenarios
│       ├── validator.go       # Main validation logic
│       ├── gitlab_client.go   # GitLab API client wrapper
│       ├── testutil/          # Test utilities and scenario discovery
│       └── *_test.go         # Validation and scenario tests
├── internal/config/            # Internal configuration management
├── test/                       # Test files and scenarios
│   ├── fixtures/              # Simple test YAML files
│   ├── simple-refactoring-cases/  # Paired before/after test files
│   ├── refactoring-scenarios/     # Complex multi-file scenarios
│   │   └── scenario-*/           # Each scenario directory
│   │       ├── before/           # Original configuration
│   │       ├── after/            # Refactored configuration
│   │       └── config.yaml       # Test expectations
│   └── realistic-app-scenarios/   # Real-world application examples
│       └── flask-microservice/   # Complete Flask app CI/CD
├── implementation-state.json      # Streamlined implementation status
└── pipeline-emulator-spec.md     # Detailed technical specification
```

### Core Components

1. **Parser Module** (`pkg/parser/`): Parses GitLab CI YAML and builds dependency graphs
2. **Differ Module** (`pkg/differ/`): Performs semantic comparison between configurations
3. **Analyzer Module** (`pkg/analyzer/`): Static analysis for common issues and optimizations
4. **Deployer Module** (`pkg/deployer/`): Manages local GitLab instance deployment
5. **Renderer Module** (`pkg/renderer/`): Renders and compares pipeline executions with visual diagram support

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

## Test Patterns

- **Simple refactoring cases**: Paired `*-before.yml` and `*-after.yml` files in `test/simple-refactoring-cases/`
- **Complex scenarios**: Directories with `before/` and `after/` subdirectories in `test/refactoring-scenarios/`
- **Include files**: Should be in `after/ci/` or `before/ci/` relative to main `.gitlab-ci.yml`
- **Test expectations**: Defined in `config.yaml` for each scenario

## Development Workflow

**IMPORTANT**: Always run after any code changes:
```bash
go test ./... && go fmt ./... && go vet ./...
```
