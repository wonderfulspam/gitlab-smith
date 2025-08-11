package parser

import (
	"os"
	"testing"
)

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
