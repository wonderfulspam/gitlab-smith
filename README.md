# GitLabSmith

GitLab CI/CD configuration analysis and validation tool.

## Quick Start

```bash
go build -o gitlab-smith ./cmd/gitlab-smith

# Analyze your .gitlab-ci.yml
gitlab-smith analyze .gitlab-ci.yml

# Compare configurations
gitlab-smith refactor --old before.yml --new after.yml
```

## Commands

```bash
# Parse configuration
gitlab-smith parse .gitlab-ci.yml

# Static analysis (72+ rules)
gitlab-smith analyze .gitlab-ci.yml

# Compare configurations  
gitlab-smith refactor --old old.yml --new new.yml

# With GitLab API validation
gitlab-smith refactor --old old.yml --new new.yml \
  --full-test --gitlab-url https://gitlab.com --gitlab-token $TOKEN

# Visualize pipeline
gitlab-smith visualize .gitlab-ci.yml --format mermaid
```

## Modes

- **Static** (default): Works offline, no GitLab needed
- **API**: Validates via GitLab API (requires token)
- **Full**: API + pipeline execution testing

## GitLab Setup (Optional)

Use GitLab.com, self-hosted, or local Docker:

```bash
# docker-compose.yml
version: '3.8'
services:
  gitlab:
    image: gitlab/gitlab-ce:latest
    ports: ["8080:80"]
    environment:
      GITLAB_ROOT_PASSWORD: password123
      EXTERNAL_URL: http://localhost:8080
```

## Features

- ✅ GitLab CI parsing with includes
- ✅ 72+ static analysis rules  
- ✅ Semantic configuration comparison
- ✅ Pipeline visualization (Mermaid/DOT)
- ✅ GitLab API integration
- 🚧 Real GitLab API client
- ⏳ Performance benchmarking

## Development

```bash
go test ./... && go fmt ./... && go vet ./...
```