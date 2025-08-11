package parser

import (
	"testing"
)

func TestParseTemplates(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "single template",
			yaml: `
stages:
  - build

.template:
  image: alpine
  script:
    - echo test
`,
			expected: 1,
		},
		{
			name: "multiple templates",
			yaml: `
stages:
  - build
  - test

.base:
  image: alpine

.build:
  extends: .base
  stage: build

.test:
  extends: .base
  stage: test

job1:
  extends: .build
  script:
    - echo build
`,
			expected: 4, // 3 templates + 1 job
		},
		{
			name: "template inheritance chain",
			yaml: `
.base:
  image: alpine
  before_script:
    - apk add git

.node_base:
  extends: .base
  image: node:18
  before_script:
    - npm install

.build_base:
  extends: .node_base
  stage: build
  before_script:
    - npm ci
    - npm run build
`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(config.Jobs) != tt.expected {
				t.Errorf("Parse() found %d jobs, expected %d", len(config.Jobs), tt.expected)
				t.Logf("Jobs found: %v", getJobNames(config.Jobs))
			}
		})
	}
}

func TestParseExtends(t *testing.T) {
	yaml := `
.parent:
  image: alpine

job:
  extends: .parent
  script:
    - echo test
`

	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	job := config.Jobs["job"]
	if job == nil {
		t.Fatal("job not found")
	}

	extends := job.GetExtends()
	if len(extends) != 1 || extends[0] != ".parent" {
		t.Errorf("job.GetExtends() = %v, expected [.parent]", extends)
	}
}

func TestParseComplexYAML(t *testing.T) {
	yaml := `
stages:
  - validate
  - build
  - test

variables:
  NODE_VERSION: "18"

.base_job:
  image: alpine:latest
  before_script:
    - apk add --no-cache git curl

.node_base:
  extends: .base_job
  image: node:18
  before_script:
    - node --version
    - npm --version

validate:yaml:
  stage: validate
  image: alpine
  script:
    - echo "validating"

build:frontend:
  extends: .node_base
  stage: build
  script:
    - npm run build

test:unit:
  extends: .node_base
  stage: test
  script:
    - npm test
`

	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expectedJobs := []string{".base_job", ".node_base", "validate:yaml", "build:frontend", "test:unit"}
	if len(config.Jobs) != len(expectedJobs) {
		t.Errorf("Parse() found %d jobs, expected %d", len(config.Jobs), len(expectedJobs))
		t.Logf("Jobs found: %v", getJobNames(config.Jobs))
		t.Logf("Expected: %v", expectedJobs)
	}

	// Test specific job parsing
	validateJob := config.Jobs["validate:yaml"]
	if validateJob == nil {
		t.Error("validate:yaml job not found")
	} else if validateJob.Stage != "validate" {
		t.Errorf("validate:yaml stage = %s, expected validate", validateJob.Stage)
	}

	// Test template parsing
	nodeBase := config.Jobs[".node_base"]
	if nodeBase == nil {
		t.Error(".node_base template not found")
	} else {
		extends := nodeBase.GetExtends()
		if len(extends) == 0 || extends[0] != ".base_job" {
			t.Errorf(".node_base.GetExtends() = %v, expected [.base_job]", extends)
		}
	}
}

func TestIsJobDefinition(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name:     "valid job with script",
			value:    map[string]interface{}{"script": []string{"echo test"}},
			expected: true,
		},
		{
			name:     "valid template with extends",
			value:    map[string]interface{}{"extends": ".parent"},
			expected: true,
		},
		{
			name:     "valid job with stage",
			value:    map[string]interface{}{"stage": "build"},
			expected: true,
		},
		{
			name:     "invalid - just a string",
			value:    "not a job",
			expected: false,
		},
		{
			name:     "invalid - empty map",
			value:    map[string]interface{}{},
			expected: false,
		},
		{
			name:     "invalid - random keys",
			value:    map[string]interface{}{"random": "value"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJobDefinition(tt.value)
			if result != tt.expected {
				t.Errorf("isJobDefinition() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Helper function for debugging
func getJobNames(jobs map[string]*JobConfig) []string {
	var names []string
	for name := range jobs {
		names = append(names, name)
	}
	return names
}
