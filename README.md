# GitLabSmith

GitLab CI/CD configuration refactoring and validation tool for safe pipeline transformations.

## Quick Start

```bash
# Build and run
go build -o gitlab-smith ./cmd/gitlab-smith
./gitlab-smith parse .gitlab-ci.yml

# Analyze a configuration
./gitlab-smith analyze .gitlab-ci.yml

# Compare configurations 
./gitlab-smith refactor --old before.yml --new after.yml
```

## Status: Phase 1 Complete, Phase 2 Partial 🚧

- ✅ **Parser**: Full GitLab CI YAML parsing with includes (local/remote/template/project)
- ✅ **Analyzer**: 72+ static analysis rules for optimization detection
- ✅ **Differ**: Semantic comparison between configurations 
- ✅ **Renderer**: Visual pipeline diagrams (Mermaid & DOT formats)
- 🚧 **Deployer**: GitLab deploys but not used for actual pipeline rendering
- ⏳ **Phase 3**: Performance benchmarking (not started)

## Commands

### Parse Configuration
```bash
./gitlab-smith parse .gitlab-ci.yml        # JSON output with job dependencies
```

### Static Analysis  
```bash
./gitlab-smith analyze .gitlab-ci.yml      # Detect optimization opportunities
```

### Compare Configurations
```bash
./gitlab-smith refactor --old before.yml --new after.yml           # Semantic comparison
./gitlab-smith refactor --old before.yml --new after.yml --full-test # With GitLab deployment
```

### Generate Visualizations
```bash
./gitlab-smith visualize .gitlab-ci.yml --format mermaid           # Mermaid flowchart
./gitlab-smith visualize .gitlab-ci.yml --format dot --output file.dot # GraphViz DOT format
```

## Development

```bash
go build -o gitlab-smith ./cmd/gitlab-smith
go test ./... && go fmt ./... && go vet ./...  # Always run after changes
```

See [CLAUDE.md](CLAUDE.md) for full development guide.