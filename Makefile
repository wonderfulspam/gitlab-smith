.PHONY: all test lint fmt vet security build clean install ci-local help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet
BINARY_NAME=gitlab-smith
MAIN_PATH=./cmd/gitlab-smith

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

# Default target
all: ci-local

# Run all CI checks locally (matches GitHub Actions)
ci-local: fmt-check vet-check lint test build
	@echo "$(GREEN)✓ All CI checks passed!$(NC)"

# Test
test:
	@echo "$(YELLOW)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Lint with golangci-lint
lint:
	@echo "$(YELLOW)Running linter...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)" && exit 1)
	golangci-lint run --timeout=10m
	@echo "$(GREEN)✓ Linting passed$(NC)"

# Format code
fmt:
	@echo "$(YELLOW)Formatting code...$(NC)"
	$(GOFMT) -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Check formatting (CI mode)
fmt-check:
	@echo "$(YELLOW)Checking formatting...$(NC)"
	@gofmt_files=$$($(GOFMT) -l .); \
	if [ -n "$$gofmt_files" ]; then \
		echo "$(RED)The following files need formatting:$(NC)"; \
		echo "$$gofmt_files"; \
		echo "Run 'make fmt' to fix"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ Format check passed$(NC)"

# Vet
vet:
	@echo "$(YELLOW)Running go vet...$(NC)"
	$(GOVET) ./...
	@echo "$(GREEN)✓ Vet passed$(NC)"

# Check vet (same as vet but for consistency)
vet-check: vet

# Security scan with gosec
security:
	@echo "$(YELLOW)Running security scan...$(NC)"
	@which gosec > /dev/null || (echo "$(RED)gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)" && exit 1)
	gosec -no-fail -fmt text ./...
	@echo "$(GREEN)✓ Security scan passed$(NC)"

# Check go mod tidy
mod-check:
	@echo "$(YELLOW)Checking go.mod...$(NC)"
	$(GOMOD) tidy
	@git diff --exit-code go.mod go.sum || (echo "$(RED)go.mod/go.sum need updating. Run 'go mod tidy'$(NC)" && exit 1)
	@echo "$(GREEN)✓ go.mod check passed$(NC)"

# Build
build:
	@echo "$(YELLOW)Building binary...$(NC)"
	$(GOBUILD) -v -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Build successful$(NC)"

# Build for multiple platforms
build-all:
	@echo "$(YELLOW)Building for all platforms...$(NC)"
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "$(GREEN)✓ All builds successful$(NC)"

# Clean
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.out results.sarif
	@echo "$(GREEN)✓ Cleaned$(NC)"

# Install
install:
	@echo "$(YELLOW)Installing...$(NC)"
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)$(NC)"

# Install CI tools
install-tools:
	@echo "$(YELLOW)Installing CI tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

# Validate gold standard cases
validate-gold:
	@echo "$(YELLOW)Validating gold standard cases...$(NC)"
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@for file in test/gold-standard-cases/*.yml; do \
		echo "Analyzing: $$file"; \
		./$(BINARY_NAME) analyze "$$file" --format json | jq -e '.issues | length == 0' > /dev/null || \
		(echo "$(RED)Gold standard case $$file produced issues$(NC)" && ./$(BINARY_NAME) analyze "$$file" && exit 1); \
	done
	@echo "$(GREEN)✓ Gold standard validation passed$(NC)"

# Test refactoring scenarios
test-scenarios:
	@echo "$(YELLOW)Testing refactoring scenarios...$(NC)"
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@for dir in test/refactoring-scenarios/*/; do \
		if [ -d "$$dir/before" ] && [ -d "$$dir/after" ]; then \
			echo "Testing scenario: $$(basename $$dir)"; \
			./$(BINARY_NAME) refactor \
				--old "$$dir/before/.gitlab-ci.yml" \
				--new "$$dir/after/.gitlab-ci.yml" \
				--format json > /dev/null || true; \
		fi \
	done
	@echo "$(GREEN)✓ Scenario tests completed$(NC)"

# Run GitHub Actions locally with act
act-ci:
	@echo "$(YELLOW)Running CI workflow locally with act...$(NC)"
	@which act > /dev/null || (echo "$(RED)act not installed. See: https://github.com/nektos/act$(NC)" && exit 1)
	act push --workflows .github/workflows/ci.yml

act-release:
	@echo "$(YELLOW)Running release workflow locally with act (dry-run)...$(NC)"
	@which act > /dev/null || (echo "$(RED)act not installed. See: https://github.com/nektos/act$(NC)" && exit 1)
	act push --workflows .github/workflows/release.yml --dry-run

# Help
help:
	@echo "GitLabSmith Makefile targets:"
	@echo "  $(YELLOW)ci-local$(NC)       - Run all CI checks locally (default)"
	@echo "  $(YELLOW)test$(NC)           - Run tests with coverage"
	@echo "  $(YELLOW)lint$(NC)           - Run golangci-lint"
	@echo "  $(YELLOW)fmt$(NC)            - Format code"
	@echo "  $(YELLOW)fmt-check$(NC)      - Check formatting"
	@echo "  $(YELLOW)vet$(NC)            - Run go vet"
	@echo "  $(YELLOW)security$(NC)       - Run gosec security scan"
	@echo "  $(YELLOW)mod-check$(NC)      - Check go.mod tidiness"
	@echo "  $(YELLOW)build$(NC)          - Build binary"
	@echo "  $(YELLOW)build-all$(NC)      - Build for all platforms"
	@echo "  $(YELLOW)clean$(NC)          - Clean build artifacts"
	@echo "  $(YELLOW)install$(NC)        - Install to GOPATH/bin"
	@echo "  $(YELLOW)install-tools$(NC)  - Install CI tools"
	@echo "  $(YELLOW)validate-gold$(NC)  - Validate gold standard cases"
	@echo "  $(YELLOW)test-scenarios$(NC) - Test refactoring scenarios"
	@echo "  $(YELLOW)act-ci$(NC)         - Test CI workflow with act"
	@echo "  $(YELLOW)act-release$(NC)    - Test release workflow with act"