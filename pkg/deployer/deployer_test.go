package deployer

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
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

func TestCheckDockerAvailability(t *testing.T) {
	deployer := New(nil)

	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	t.Run("docker available", func(t *testing.T) {
		if _, err := exec.LookPath("docker"); err != nil {
			t.Skip("Docker not available in PATH, skipping test")
		}

		err := deployer.checkDockerAvailability()
		if err != nil && !strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Errorf("Expected success or daemon connection error, got: %v", err)
		}
	})

	t.Run("docker not available", func(t *testing.T) {
		os.Setenv("PATH", "")

		err := deployer.checkDockerAvailability()
		if err == nil {
			t.Error("Expected error when docker not available")
		}

		if !strings.Contains(err.Error(), "docker is not available") {
			t.Errorf("Expected 'docker is not available' error, got: %v", err)
		}
	})
}

func TestCreateDataDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &DeploymentConfig{
		DataVolumePath:   tmpDir + "/data",
		ConfigVolumePath: tmpDir + "/config", 
		LogsVolumePath:   tmpDir + "/logs",
	}

	deployer := New(config)
	
	err := deployer.createDataDirectories()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that directories were created
	dirs := []string{config.DataVolumePath, config.ConfigVolumePath, config.LogsVolumePath}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestCreateDataDirectoriesError(t *testing.T) {
	config := &DeploymentConfig{
		DataVolumePath: "/root/readonly/data", // Should fail on most systems
	}

	deployer := New(config)
	
	err := deployer.createDataDirectories()
	if err == nil {
		t.Error("Expected error when creating directory in readonly location")
	}
}

func TestCleanup(t *testing.T) {
	deployer := New(nil)
	
	err := deployer.cleanup()
	if err != nil {
		t.Errorf("Cleanup should not return error even if container doesn't exist, got: %v", err)
	}
}

func TestGetLogsToBuffer(t *testing.T) {
	deployer := New(nil)
	var buf bytes.Buffer
	
	err := deployer.GetLogs(&buf, false)
	if err == nil {
		t.Error("Expected error when getting logs from non-existent container")
	}
}

func TestGetStatusNonExistentContainer(t *testing.T) {
	config := &DeploymentConfig{
		ContainerName:    "non-existent-container-test-12345",
		ExternalHostname: "localhost",
		HTTPPort:         "8080",
	}
	
	deployer := New(config)
	
	status, err := deployer.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus should not return error, got: %v", err)
	}
	
	if status.IsRunning {
		t.Error("Expected IsRunning to be false for non-existent container")
	}
	
	if status.ContainerName != config.ContainerName {
		t.Errorf("Expected ContainerName %s, got %s", config.ContainerName, status.ContainerName)
	}
	
	expectedURL := "http://localhost:8080"
	if status.URL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, status.URL)
	}
}
