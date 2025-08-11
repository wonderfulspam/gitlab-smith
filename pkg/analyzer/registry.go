package analyzer

import (
	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

// Checker interface for all check functions
type Checker interface {
	Check(config *parser.GitLabConfig) []types.Issue
	Name() string
	Type() types.IssueType
	Enabled() bool
}

// CheckRegistry manages all available checks
type CheckRegistry struct {
	checks map[string]Checker
}

func NewCheckRegistry() *CheckRegistry {
	return &CheckRegistry{
		checks: make(map[string]Checker),
	}
}

func (r *CheckRegistry) Register(name string, issueType types.IssueType, checkFunc types.CheckFunc) {
	checker := NewBaseChecker(name, issueType, checkFunc)
	r.checks[name] = checker
}

func (r *CheckRegistry) GetChecks() []Checker {
	checks := make([]Checker, 0, len(r.checks))
	for _, check := range r.checks {
		checks = append(checks, check)
	}
	return checks
}

func (r *CheckRegistry) GetChecksByType(issueType types.IssueType) []Checker {
	checks := make([]Checker, 0)
	for _, check := range r.checks {
		if check.Type() == issueType {
			checks = append(checks, check)
		}
	}
	return checks
}

// BaseChecker provides common functionality for all checkers
type BaseChecker struct {
	name        string
	issueType   types.IssueType
	enabled     bool
	checkFunc   types.CheckFunc
	description string
}

func NewBaseChecker(name string, issueType types.IssueType, checkFunc types.CheckFunc) *BaseChecker {
	return &BaseChecker{
		name:      name,
		issueType: issueType,
		enabled:   true,
		checkFunc: checkFunc,
	}
}

func (c *BaseChecker) Check(config *parser.GitLabConfig) []types.Issue {
	if !c.enabled {
		return []types.Issue{}
	}
	return c.checkFunc(config)
}

func (c *BaseChecker) Name() string {
	return c.name
}

func (c *BaseChecker) Type() types.IssueType {
	return c.issueType
}

func (c *BaseChecker) Enabled() bool {
	return c.enabled
}

func (c *BaseChecker) SetEnabled(enabled bool) {
	c.enabled = enabled
}

func (c *BaseChecker) SetDescription(description string) {
	c.description = description
}
