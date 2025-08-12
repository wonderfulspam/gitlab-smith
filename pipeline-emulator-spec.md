# GitLabSmith - GitLab CI Refactoring and Validation Framework

## Project Overview

**GitLabSmith** is a local development tool for refactoring and validating GitLab CI/CD configurations. It helps developers safely transform complex monorepo CI definitions by providing semantic diffing, behavioral validation through local GitLab deployment, and performance testing in controlled environments.

## Core Objectives

### Primary Use Cases
1. **CI Refactoring**: Transform dozen-file CI definitions into maintainable structures
2. **Rule Simplification**: Clean up complex `changes:` and conditional logic with confidence
3. **Equivalence Validation**: Prove refactored pipelines behave identically via local GitLab testing
4. **Performance Testing**: Validate PVC vs artifact approach with ad-hoc infrastructure
5. **Configuration Analysis**: Static analysis and optimization suggestions

### Modular Capabilities
- **Semantic Diffing**: GitLab CI-aware comparison that understands inheritance and includes
- **Static Analysis**: Dependency mapping, dead code detection, rule optimization
- **Local Testing**: Deploy temporary GitLab instance for complete behavioral validation
- **Performance Benchmarking**: Compare startup times in controlled test environment
- **Helm Chart Generation**: Convert runner configs to maintainable Helm charts

## Architecture Overview

GitLabSmith provides **two modes** depending on your validation needs:

**Mode 1: Static Analysis (Production GitLab API)**
```
┌─────────────────────────────────────────────────────────────┐
│                 GitLabSmith Static Mode                     │
├─────────────────────────────────────────────────────────────┤
│  Local CI Parser & Analyzer  │  Production GitLab API      │
│  Semantic Diff Engine        │  (Include resolution only)  │
└─────────────────────────────────────────────────────────────┘
           │                              │
┌──────────▼──────────────┐    ┌─────────▼─────────┐
│   Static Analysis       │    │   Limited API     │
│ • Dependency mapping    │    │ • Merged YAML     │
│ • Rule optimization     │    │ • Basic validation│
│ • Dead code detection   │    │ • Include resolution│
└─────────────────────────┘    └───────────────────┘
```

**Mode 2: Full Behavioral Testing (Local GitLab)**
```
┌─────────────────────────────────────────────────────────────┐
│                GitLabSmith Full Mode                        │
├─────────────────────────────────────────────────────────────┤
│  All Static Analysis     │  Local GitLab Infrastructure    │
│  + Behavioral Testing    │  (Complete control)              │
└─────────────────────────────────────────────────────────────┘
           │                              │
┌──────────▼──────────────┐    ┌─────────▼─────────┐
│   Complete Validation   │    │  Test Environment │
│ • Changes rule testing  │    │ • Local GitLab    │
│ • Pipeline equivalence  │    │ • Test runners    │
│ • Performance benchmarks│    │ • PVC validation  │
└─────────────────────────┘    └───────────────────┘
```

## Implementation Plan

### Phase 1: Core Analysis & Semantic Diffing (Claude Code Ready)
**Goal**: Build CLI for parsing and comparing GitLab CI configurations

**Features**:
- **Local CI Parser**: Parse dozen-file CI configs with dependency mapping
- **Semantic Differ**: GitLab CI-aware comparison using production API's `merged_yaml`
- **Rule Analysis**: Static analysis of `changes:` patterns and complexity detection
- **Dead Code Detection**: Identify unused jobs, variables, and redundant rules

**What you get**:
- Parse and understand your complex CI structure locally
- Generate semantic diffs showing real behavioral changes vs cosmetic changes
- Identify optimization opportunities and unused configuration
- Basic include resolution via production GitLab API for accurate comparison

**Limitations**:
- Cannot test `changes:` rules with actual file changes
- Cannot see fully rendered pipeline structure
- Static analysis only - no behavioral validation

### Phase 2: Local GitLab for Pipeline Rendering
**Goal**: Deploy local GitLab to see complete pipeline structure and validation

**Infrastructure**:
- **Local GitLab Instance**: Docker-based GitLab with your project mirrored
- **Pipeline Visualization**: See fully rendered pipelines with all rules evaluated
- **Include Resolution**: Complete local resolution of all includes and templates
- **Advanced Validation**: Full GitLab CI lint and validation capabilities

**New Capabilities**:
- **Complete Pipeline Rendering**: See exactly what jobs would be created
- **Rule Evaluation**: Understand how complex `changes:` and `if:` rules resolve
- **Template Testing**: Test includes and extends without affecting production
- **Configuration Validation**: Full GitLab validation including enterprise features

### Phase 3: Runner Deployment & Performance Testing
**Goal**: Add runners for behavioral equivalence and performance validation

**Infrastructure Addition**:
- **Test Runners**: Kubernetes runners with PVC and artifact configurations
- **Automated Testing**: Create test commits to validate `changes:` behavior
- **Performance Benchmarking**: Real startup time comparisons
- **Load Testing**: 100+ parallel job execution testing

**Complete Capabilities**:
- **Changes Rule Validation**: Create test commits to validate `changes:` behavior
- **Behavioral Equivalence**: Compare actual pipeline execution between configs
- **PVC Performance Testing**: Quantify startup improvements across many jobs
- **Complete Confidence**: Proof that refactored configs behave identically

## Core Commands & Workflows

```bash
# Static analysis mode (works with production GitLab API)
gitlabsmith config set --gitlab-url=https://gitlab.company.com --token=$GITLAB_TOKEN
gitlabsmith parse --project=myorg/monorepo --analyze-rules
gitlabsmith diff --before=.gitlab-ci-old.yml --after=.gitlab-ci-new.yml --semantic

# Full validation mode (requires local GitLab)
gitlabsmith infra deploy --mirror-project=myorg/monorepo
gitlabsmith test changes --rule="src/api/*" --files="src/api/new.go,src/web/old.js"
gitlabsmith validate equivalence --original=old/ --refactored=new/ --prove-behavior
gitlabsmith benchmark --pvc-vs-artifacts --jobs=100 --nodes=12
```

## Technology Stack

### Core Framework
- **Language**: Go for performance and single-binary distribution
- **CLI**: Cobra for robust command structure and help system
- **YAML Processing**: goccy/go-yaml + dyff integration for semantic diffing
- **GitLab Integration**: Production GitLab API for include resolution only
- **Configuration**: Local YAML config for GitLab connection details

### Local Infrastructure (Full Mode Only)
- **Local GitLab**: Docker-based GitLab instance for behavioral testing
- **Test Runners**: Kubernetes runners for PVC vs artifact benchmarking
- **Performance Testing**: Only possible with local infrastructure you control

## Configuration Management

### Overview
GitLabSmith provides a flexible configuration system that allows users to customize analysis behavior, severity levels, and filtering rules to match their organization's specific needs and CI/CD practices.

### Configuration File Format
Configuration is stored in YAML or JSON format (`.gitlab-smith.yml` or `.gitlab-smith.json`):

```yaml
# .gitlab-smith.yml
version: "1.0"
analyzer:
  # Global severity threshold - only report issues at or above this level
  severity_threshold: low  # low, medium, high
  # Configure individual checks
  checks:
    job_naming:
      enabled: true
      severity: low  # Override default severity
      ignore_patterns:
        - "legacy-*"  # Ignore legacy job names
        - "*-deprecated"
      exclusions:
        jobs:
          - "my job with spaces"  # Specific job to ignore
          - "another legacy job"
    image_tags:
      enabled: true
      severity: high  # Elevate importance
      allowed_tags:
        - "latest"  # Sometimes needed for internal images
        - "stable"
    script_complexity:
      enabled: true
      max_lines: 50  # Custom threshold
      max_commands: 20
    cache_usage:
      enabled: true
      required_for_stages:
        - build
        - test

  # Pattern-based exclusions across all checks
  global_exclusions:
    paths:
      - "experimental/*"
      - "third-party/*"
    jobs:
      - "*-experimental"
      - "sandbox-*"

# Differ configuration
differ:
  # Ignore certain types of changes
  ignore_changes:
    - variable_order  # Don't report reordered variables
    - comment_changes  # Ignore comment-only changes

  # Treat certain changes as improvements
  improvement_patterns:
    - consolidation  # Combining duplicate jobs
    - simplification  # Reducing rule complexity
# Output configuration
output:
  format: table  # table, json, yaml
  verbose: false
  show_suggestions: true
  group_by: type  # type, severity, job
```

### Use Cases

#### 1. Ignoring Specific Findings
Users can ignore findings for specific jobs, patterns, or paths:
- **Job-specific exclusions**: Whitelist specific jobs that don't follow conventions
- **Pattern-based ignoring**: Use glob patterns to exclude groups of jobs
- **Path-based exclusions**: Ignore issues in certain directories

#### 2. Adjusting Severity Levels
Override default severity for any check to match organizational priorities:
- Elevate security checks to high severity
- Reduce maintainability checks to low severity for legacy code
- Set global thresholds to filter noise

#### 3. Custom Thresholds
Configure check-specific parameters:
- Maximum script complexity limits
- Required cache configuration for specific stages
- Allowed Docker image tags

#### 4. Global Filtering
Apply organization-wide rules:
- Exclude experimental or third-party configurations
- Set minimum severity for CI/CD pipeline failures
- Configure output preferences

### Configuration Precedence
1. CLI flags (highest priority)
2. Environment variables (`GITLAB_SMITH_*`)
3. Project configuration file (`.gitlab-smith.yml`)
4. User configuration file (`~/.gitlab-smith/config.yml`)
5. System defaults (lowest priority)

### CLI Integration
```bash
# Use specific config file
gitlabsmith analyze --config=custom-config.yml

# Override severity threshold
gitlabsmith analyze --severity-threshold=high

# Disable specific check
gitlabsmith analyze --disable-check=job_naming

# Generate default config
gitlabsmith config init

# Validate config file
gitlabsmith config validate
```

GitLabSmith provides the tooling needed to safely refactor complex GitLab CI configurations - from static analysis that works immediately, to full behavioral validation when you need absolute confidence in your changes.