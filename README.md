# GitLabSmith

A GitLab CI/CD configuration refactoring and validation tool for safe pipeline transformations.

## Quick Start

```bash
# Parse a GitLab CI file
./gitlab-smith parse .gitlab-ci.yml

# Build from source
go build -o gitlab-smith ./cmd/gitlab-smith
```

## Current Status

**Phase 1: Core Parser** ✅ (Current implementation)
- GitLab CI YAML parsing with dependency mapping
- JSON output for analysis and integration
- Command-line interface for file parsing

**Phase 2: Semantic Analysis** (Next)
- Configuration comparison and diffing
- Static analysis with optimization suggestions

**Phase 3: Local Testing** (Future)
- Local GitLab deployment for behavioral validation
- Performance benchmarking and comparison

## Installation

### Prerequisites
- Go 1.21+ 
- Unix/Linux environment

### From Source
```bash
git clone <repository-url>
cd gitlab-smith
go build -o gitlab-smith ./cmd/gitlab-smith
```

## Usage

### Parse GitLab CI Configuration
```bash
./gitlab-smith parse path/to/.gitlab-ci.yml
```

Outputs structured JSON representation of the GitLab CI configuration including:
- Job definitions with dependencies
- Stage configurations
- Variable definitions
- Include directives

## Documentation

- **[CLAUDE.md](CLAUDE.md)** - Development guide for contributors and Claude Code
- **[pipeline-emulator-spec.md](pipeline-emulator-spec.md)** - Complete technical specification and architecture

## Contributing

See [CLAUDE.md](CLAUDE.md) for development setup and project structure.

## License

TBD