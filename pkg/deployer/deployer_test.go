package deployer

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.ContainerName != "gitlab-smith-test" {
		t.Errorf("Expected ContainerName to be 'gitlab-smith-test', got '%s'", config.ContainerName)
	}

	if config.GitLabImage != "gitlab/gitlab-ce:latest" {
		t.Errorf("Expected GitLabImage to be 'gitlab/gitlab-ce:latest', got '%s'", config.GitLabImage)
	}

	if config.HTTPPort != "8080" {
		t.Errorf("Expected HTTPPort to be '8080', got '%s'", config.HTTPPort)
	}

	if config.SSHPort != "2222" {
		t.Errorf("Expected SSHPort to be '2222', got '%s'", config.SSHPort)
	}
}

func TestNew(t *testing.T) {
	t.Run("with custom config", func(t *testing.T) {
		config := &DeploymentConfig{
			ContainerName: "test-container",
			GitLabImage:   "gitlab/gitlab-ce:13.12.0",
		}

		deployer := New(config)

		if deployer == nil {
			t.Fatal("New returned nil")
		}

		if deployer.config.ContainerName != "test-container" {
			t.Errorf("Expected ContainerName to be 'test-container', got '%s'", deployer.config.ContainerName)
		}

		if deployer.config.GitLabImage != "gitlab/gitlab-ce:13.12.0" {
			t.Errorf("Expected GitLabImage to be 'gitlab/gitlab-ce:13.12.0', got '%s'", deployer.config.GitLabImage)
		}
	})

	t.Run("with nil config", func(t *testing.T) {
		deployer := New(nil)

		if deployer == nil {
			t.Fatal("New returned nil")
		}

		if deployer.config.ContainerName != "gitlab-smith-test" {
			t.Errorf("Expected default ContainerName to be 'gitlab-smith-test', got '%s'", deployer.config.ContainerName)
		}
	})
}

func TestDeploymentStatus(t *testing.T) {
	status := &DeploymentStatus{
		ContainerName:   "test-container",
		IsRunning:       true,
		ContainerStatus: "Up 5 minutes",
		URL:             "http://localhost:8080",
	}

	if status.ContainerName != "test-container" {
		t.Errorf("Expected ContainerName to be 'test-container', got '%s'", status.ContainerName)
	}

	if !status.IsRunning {
		t.Error("Expected IsRunning to be true")
	}

	if status.URL != "http://localhost:8080" {
		t.Errorf("Expected URL to be 'http://localhost:8080', got '%s'", status.URL)
	}
}
