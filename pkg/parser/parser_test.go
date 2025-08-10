package parser

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/simple.gitlab-ci.yml")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	config, err := Parse(data)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	if len(config.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(config.Stages))
	}

	expectedStages := []string{"build", "test", "deploy"}
	for i, stage := range expectedStages {
		if i >= len(config.Stages) || config.Stages[i] != stage {
			t.Errorf("expected stage %d to be %s, got %v", i, stage, config.Stages[i])
		}
	}

	if len(config.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(config.Variables))
	}

	if config.Default == nil {
		t.Error("expected default config to be set")
	} else {
		if config.Default.Image != "node:16" {
			t.Errorf("expected default image to be node:16, got %s", config.Default.Image)
		}
	}

	if len(config.Jobs) != 5 {
		t.Errorf("expected 5 jobs, got %d", len(config.Jobs))
	}

	buildJob, exists := config.Jobs["build"]
	if !exists {
		t.Error("expected 'build' job to exist")
	} else {
		if buildJob.Stage != "build" {
			t.Errorf("expected build job stage to be 'build', got %s", buildJob.Stage)
		}
		if buildJob.Artifacts == nil || len(buildJob.Artifacts.Paths) != 1 {
			t.Error("expected build job to have artifacts with 1 path")
		}
	}

	testUnitJob, exists := config.Jobs["test:unit"]
	if !exists {
		t.Error("expected 'test:unit' job to exist")
	} else {
		if testUnitJob.Needs == nil {
			t.Error("expected test:unit to have needs")
		} else if needsSlice, ok := testUnitJob.Needs.([]interface{}); ok {
			if len(needsSlice) != 1 || needsSlice[0] != "build" {
				t.Errorf("expected test:unit to need 'build', got %v", needsSlice)
			}
		} else {
			t.Errorf("unexpected needs type: %T", testUnitJob.Needs)
		}
	}

	deployProdJob, exists := config.Jobs["deploy:production"]
	if !exists {
		t.Error("expected 'deploy:production' job to exist")
	} else {
		if deployProdJob.When != "manual" {
			t.Errorf("expected deploy:production when to be 'manual', got %s", deployProdJob.When)
		}
		if len(deployProdJob.Rules) != 1 {
			t.Errorf("expected deploy:production to have 1 rule, got %d", len(deployProdJob.Rules))
		}
	}
}

func TestGetDependencyGraph(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/simple.gitlab-ci.yml")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	config, err := Parse(data)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	graph := config.GetDependencyGraph()

	buildDeps := graph["build"]
	if len(buildDeps) != 0 {
		t.Errorf("expected build job to have no dependencies, got %v", buildDeps)
	}

	testUnitDeps := graph["test:unit"]
	if len(testUnitDeps) != 1 || testUnitDeps[0] != "build" {
		t.Errorf("expected test:unit to depend on 'build', got %v", testUnitDeps)
	}

	testIntegrationDeps := graph["test:integration"]
	if len(testIntegrationDeps) != 1 || testIntegrationDeps[0] != "build" {
		t.Errorf("expected test:integration to depend on 'build', got %v", testIntegrationDeps)
	}

	deployStagingDeps := graph["deploy:staging"]
	if len(deployStagingDeps) != 2 {
		t.Errorf("expected deploy:staging to have 2 dependencies, got %v", deployStagingDeps)
	}

	deployProdDeps := graph["deploy:production"]
	if len(deployProdDeps) != 2 {
		t.Errorf("expected deploy:production to have 2 dependencies, got %v", deployProdDeps)
	}
}

func TestParseMinimal(t *testing.T) {
	yaml := `
test:
  script:
    - echo "Hello, World!"
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing minimal config: %v", err)
	}

	if len(config.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(config.Jobs))
	}

	testJob, exists := config.Jobs["test"]
	if !exists {
		t.Error("expected 'test' job to exist")
	} else {
		if len(testJob.Script) != 1 || testJob.Script[0] != "echo \"Hello, World!\"" {
			t.Errorf("unexpected script: %v", testJob.Script)
		}
	}
}

func TestParseComplexNeeds(t *testing.T) {
	yaml := `
build:
  script:
    - make build

test:
  script:
    - make test
  needs:
    - job: build
      optional: true

deploy:
  script:
    - make deploy
  needs:
    - build
    - test
`
	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing config with complex needs: %v", err)
	}

	testJob := config.Jobs["test"]
	if needsSlice, ok := testJob.Needs.([]interface{}); ok {
		if len(needsSlice) != 1 {
			t.Errorf("expected test job to have 1 need, got %d", len(needsSlice))
		}
		if needMap, ok := needsSlice[0].(map[string]interface{}); ok {
			if needMap["job"] != "build" || needMap["optional"] != true {
				t.Errorf("unexpected needs for test job: %+v", needMap)
			}
		}
	} else {
		t.Errorf("unexpected needs type: %T", testJob.Needs)
	}

	deployJob := config.Jobs["deploy"]
	if needsSlice, ok := deployJob.Needs.([]interface{}); ok {
		if len(needsSlice) != 2 {
			t.Errorf("expected deploy job to have 2 needs, got %d", len(needsSlice))
		}
	}
}
