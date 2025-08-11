package analyzer

import (
	"github.com/emt/gitlab-smith/pkg/parser"
	"strings"
	"testing"
)

func TestCheckTemplateComplexity(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "simple template",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					".base": {
						Image: "alpine",
					},
				},
			},
			expected: 0, // No complexity issues
		},
		{
			name: "deep inheritance chain",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					".base": {
						Image: "alpine",
					},
					".level1": {
						Extends: ".base",
						Image:   "node",
					},
					".level2": {
						Extends: ".level1",
						Stage:   "build",
					},
					".level3": {
						Extends: ".level2",
						Stage:   "build",
					},
					".level4": {
						Extends: ".level3",
						Stage:   "build",
					},
				},
			},
			expected: 1, // Should detect deep inheritance in .level4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkTemplateComplexity(tt.config, result)

			complexityIssues := 0
			for _, issue := range result.Issues {
				if issue.Type == IssueTypeMaintainability && issue.Path == "templates..level4" {
					complexityIssues++
				}
			}

			if complexityIssues != tt.expected {
				t.Errorf("checkTemplateComplexity() = %d issues, expected %d", complexityIssues, tt.expected)
			}
		})
	}
}

func TestCheckRedundantInheritance(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "redundant before_script commands",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					".parent": {
						BeforeScript: []string{"echo parent", "npm install"},
					},
					".child": {
						Extends:      ".parent",
						BeforeScript: []string{"echo parent", "echo child"}, // "echo parent" is redundant
					},
				},
			},
			expected: 1, // Should detect redundant command
		},
		{
			name: "no redundancy",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					".parent": {
						BeforeScript: []string{"echo parent"},
					},
					".child": {
						Extends:      ".parent",
						BeforeScript: []string{"echo child"},
					},
				},
			},
			expected: 0, // No redundancy
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkRedundantInheritance(tt.config, result)

			redundancyIssues := 0
			for _, issue := range result.Issues {
				if issue.Type == IssueTypeMaintainability &&
					issue.Path == "templates..child.before_script" {
					redundancyIssues++
				}
			}

			if redundancyIssues != tt.expected {
				t.Errorf("checkRedundantInheritance() = %d issues, expected %d", redundancyIssues, tt.expected)
			}
		})
	}
}

func TestCheckMatrixOpportunities(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "similar jobs that could use matrix",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test:node14": {Stage: "test", Image: "node:14"},
					"test:node16": {Stage: "test", Image: "node:16"},
					"test:node18": {Stage: "test", Image: "node:18"},
				},
			},
			expected: 1, // Should suggest matrix strategy
		},
		{
			name: "different stages - no matrix opportunity",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"build": {Stage: "build", Image: "node:16"},
					"test":  {Stage: "test", Image: "node:16"},
				},
			},
			expected: 0, // Different stages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkMatrixOpportunities(tt.config, result)

			matrixIssues := 0
			for _, issue := range result.Issues {
				if issue.Type == IssueTypePerformance {
					matrixIssues++
				}
			}

			if matrixIssues != tt.expected {
				t.Errorf("checkMatrixOpportunities() = %d issues, expected %d", matrixIssues, tt.expected)
			}
		})
	}
}

func TestCheckIncludeOptimization(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "too many includes",
			config: &parser.GitLabConfig{
				Include: []parser.Include{
					{Local: "ci/build.yml"},
					{Local: "ci/test.yml"},
					{Local: "ci/deploy.yml"},
					{Local: "ci/security.yml"},
					{Local: "ci/lint.yml"},
					{Local: "ci/package.yml"},
				},
			},
			expected: 2, // Should suggest consolidation (many includes + many local includes)
		},
		{
			name: "reasonable includes",
			config: &parser.GitLabConfig{
				Include: []parser.Include{
					{Local: "ci/build.yml"},
					{Local: "ci/test.yml"},
				},
			},
			expected: 0, // No issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkIncludeOptimization(tt.config, result)

			includeIssues := 0
			for _, issue := range result.Issues {
				if issue.Type == IssueTypeMaintainability && issue.Path == "include" {
					includeIssues++
				}
			}

			if includeIssues != tt.expected {
				t.Errorf("checkIncludeOptimization() = %d issues, expected %d", includeIssues, tt.expected)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("calculateTemplateDepth", func(t *testing.T) {
		templates := map[string]*parser.JobConfig{
			".base":   {Image: "alpine"},
			".level1": {Extends: []string{".base"}},
			".level2": {Extends: []string{".level1"}},
		}

		depth := calculateTemplateDepth(".level2", templates, make(map[string]bool))
		if depth != 3 {
			t.Errorf("calculateTemplateDepth() = %d, expected 3", depth)
		}
	})

	t.Run("findRedundantCommands", func(t *testing.T) {
		parent := []string{"echo parent", "npm install"}
		child := []string{"echo parent", "echo child"}

		redundant := findRedundantCommands(child, parent)
		if len(redundant) != 1 || redundant[0] != "echo parent" {
			t.Errorf("findRedundantCommands() = %v, expected [\"echo parent\"]", redundant)
		}
	})

	t.Run("getTemplates", func(t *testing.T) {
		config := &parser.GitLabConfig{
			Jobs: map[string]*parser.JobConfig{
				".template":  {Image: "alpine"},
				"actual_job": {Stage: "build"},
			},
		}

		templates := getTemplates(config)
		if len(templates) != 1 || templates[".template"] == nil {
			t.Errorf("getTemplates() should return only template jobs starting with '.'")
		}
	})
}

// Tests for enhanced duplication detection

func TestCheckDuplicatedBeforeScripts_Enhanced(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected struct {
			exact   int // exact duplicates
			similar int // similar with high overlap
		}
	}{
		{
			name: "exact duplicates",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {BeforeScript: []string{"apt-get update", "apt-get install -y curl"}},
					"job2": {BeforeScript: []string{"apt-get update", "apt-get install -y curl"}},
					"job3": {BeforeScript: []string{"different", "commands"}},
				},
			},
			expected: struct {
				exact   int
				similar int
			}{exact: 1, similar: 0},
		},
		{
			name: "similar with high overlap",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"build:frontend": {BeforeScript: []string{
						"echo 'Starting frontend build...'",
						"apt-get update -qq",
						"apt-get install -y -qq git curl",
						"node --version",
						"npm --version",
						"npm ci --cache .npm --prefer-offline",
					}},
					"build:backend": {BeforeScript: []string{
						"echo 'Starting backend build...'",
						"apt-get update -qq",
						"apt-get install -y -qq git curl",
						"node --version",
						"npm --version",
						"npm ci --cache .npm --prefer-offline",
					}},
				},
			},
			expected: struct {
				exact   int
				similar int
			}{exact: 0, similar: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkDuplicatedBeforeScripts(tt.config, result)

			exactCount := 0
			similarCount := 0

			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "Duplicate before_script blocks") {
					exactCount++
				} else if strings.Contains(issue.Message, "Similar before_script blocks") {
					similarCount++
				}
			}

			if exactCount != tt.expected.exact {
				t.Errorf("Expected %d exact duplicates, got %d", tt.expected.exact, exactCount)
			}
			if similarCount != tt.expected.similar {
				t.Errorf("Expected %d similar overlaps, got %d", tt.expected.similar, similarCount)
			}
		})
	}
}

func TestCheckDuplicatedSetup_Enhanced(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected struct {
			fullSetup int
			images    int
			services  int
		}
	}{
		{
			name: "multiple images and services",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {Image: "node:16", Services: []string{"redis:6"}},
					"job2": {Image: "node:16", Services: []string{"redis:6"}},
					"job3": {Image: "node:16", Services: []string{"redis:6"}},
					"job4": {Image: "node:16"},
					"job5": {Image: "node:16"},
					"job6": {Image: "python:3.9"},
				},
			},
			expected: struct {
				fullSetup int
				images    int
				services  int
			}{fullSetup: 1, images: 1, services: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkDuplicatedSetup(tt.config, result)

			setupCount := 0
			imageCount := 0
			serviceCount := 0

			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "Duplicate setup configuration") {
					setupCount++
				} else if strings.Contains(issue.Message, "Image") && strings.Contains(issue.Message, "used in multiple jobs") {
					imageCount++
				} else if strings.Contains(issue.Message, "Services") && strings.Contains(issue.Message, "used in multiple jobs") {
					serviceCount++
				}
			}

			if setupCount != tt.expected.fullSetup {
				t.Errorf("Expected %d full setup duplicates, got %d", tt.expected.fullSetup, setupCount)
			}
			if imageCount != tt.expected.images {
				t.Errorf("Expected %d image duplications, got %d", tt.expected.images, imageCount)
			}
			if serviceCount != tt.expected.services {
				t.Errorf("Expected %d service duplications, got %d", tt.expected.services, serviceCount)
			}
		})
	}
}

func TestCheckMissingExtends(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "no templates but similar jobs",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
					"job2": {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
					"job3": {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
					"job4": {Stage: "build", Image: "python:3.9", Script: []string{"python setup.py build"}},
					"job5": {Stage: "build", Image: "python:3.9", Script: []string{"python setup.py build"}},
					"job6": {Stage: "build", Image: "python:3.9", Script: []string{"python setup.py build"}},
				},
			},
			expected: 1, // Should suggest template extraction
		},
		{
			name: "templates already exist",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					".node_template":   {Stage: "test", Image: "node:16"},
					".python_template": {Stage: "build", Image: "python:3.9"},
					"job1":             {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
					"job2":             {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
					"job3":             {Stage: "test", Image: "node:16", Script: []string{"npm test"}},
				},
			},
			expected: 0, // Templates exist, should not suggest more
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkMissingExtends(tt.config, result)

			extendsCount := 0
			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "similar jobs that could benefit from template extraction") {
					extendsCount++
				}
			}

			if extendsCount != tt.expected {
				t.Errorf("Expected %d extends suggestions, got %d", tt.expected, extendsCount)
			}
		})
	}
}

func TestCheckMissingNeeds(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected struct {
			needsOpportunities  int
			parallelizationTips int
		}
	}{
		{
			name: "many dependencies without needs",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {Dependencies: []string{"build1"}},
					"job2": {Dependencies: []string{"build2"}},
					"job3": {Dependencies: []string{"build3"}},
					"job4": {Dependencies: []string{"build4"}},
				},
			},
			expected: struct {
				needsOpportunities  int
				parallelizationTips int
			}{needsOpportunities: 1, parallelizationTips: 0},
		},
		{
			name: "stage with many parallelizable jobs",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"test1": {Stage: "test"},
					"test2": {Stage: "test"},
					"test3": {Stage: "test"},
					"test4": {Stage: "test"},
					"test5": {Stage: "test"},
				},
			},
			expected: struct {
				needsOpportunities  int
				parallelizationTips int
			}{needsOpportunities: 0, parallelizationTips: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkMissingNeeds(tt.config, result)

			needsCount := 0
			parallelCount := 0

			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "could benefit from 'needs'") {
					needsCount++
				} else if strings.Contains(issue.Message, "could potentially run in parallel") {
					parallelCount++
				}
			}

			if needsCount != tt.expected.needsOpportunities {
				t.Errorf("Expected %d needs opportunities, got %d", tt.expected.needsOpportunities, needsCount)
			}
			if parallelCount != tt.expected.parallelizationTips {
				t.Errorf("Expected %d parallelization tips, got %d", tt.expected.parallelizationTips, parallelCount)
			}
		})
	}
}

func TestCheckMissingTemplates(t *testing.T) {
	tests := []struct {
		name     string
		config   *parser.GitLabConfig
		expected int
	}{
		{
			name: "repeated before_script patterns",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {BeforeScript: []string{"apt-get update", "apt-get install curl"}},
					"job2": {BeforeScript: []string{"apt-get update", "apt-get install curl"}},
					"job3": {BeforeScript: []string{"apt-get update", "apt-get install curl"}},
				},
			},
			expected: 1, // Should suggest template for repeated before_script
		},
		{
			name: "repeated setup patterns",
			config: &parser.GitLabConfig{
				Jobs: map[string]*parser.JobConfig{
					"job1": {Image: "node:16", Services: []string{"redis:6"}},
					"job2": {Image: "node:16", Services: []string{"redis:6"}},
					"job3": {Image: "node:16", Services: []string{"redis:6"}},
				},
			},
			expected: 1, // Should suggest template for repeated setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{Issues: []Issue{}}
			checkMissingTemplates(tt.config, result)

			templateCount := 0
			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "could benefit from template extraction") {
					templateCount++
				}
			}

			if templateCount != tt.expected {
				t.Errorf("Expected %d template suggestions, got %d", tt.expected, templateCount)
			}
		})
	}
}

func TestCalculateScriptOverlap(t *testing.T) {
	tests := []struct {
		name     string
		script1  []string
		script2  []string
		expected float64
	}{
		{
			name:     "identical scripts",
			script1:  []string{"apt-get update", "npm install"},
			script2:  []string{"apt-get update", "npm install"},
			expected: 1.0,
		},
		{
			name:     "no overlap",
			script1:  []string{"echo hello"},
			script2:  []string{"echo world"},
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			script1:  []string{"apt-get update", "apt-get install curl", "npm install"},
			script2:  []string{"apt-get update", "apt-get install curl", "python setup.py install"},
			expected: 0.67, // 2/3 overlap
		},
		{
			name:     "high overlap with case differences",
			script1:  []string{"APT-GET UPDATE", "npm install"},
			script2:  []string{"apt-get update", "npm install"},
			expected: 1.0, // Case insensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateScriptOverlap(tt.script1, tt.script2)

			// Allow small floating point differences
			if abs(result-tt.expected) > 0.01 {
				t.Errorf("Expected overlap %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
