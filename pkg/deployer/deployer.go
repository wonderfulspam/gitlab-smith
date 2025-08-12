package deployer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DeploymentConfig represents the configuration for GitLab deployment
type DeploymentConfig struct {
	ContainerName    string
	GitLabImage      string
	ExternalHostname string
	HTTPPort         string
	SSHPort          string
	RootPassword     string
	DataVolumePath   string
	ConfigVolumePath string
	LogsVolumePath   string
}

// DefaultConfig returns a default deployment configuration
func DefaultConfig() *DeploymentConfig {
	return &DeploymentConfig{
		ContainerName:    "gitlab-smith-test",
		GitLabImage:      "gitlab/gitlab-ce:latest",
		ExternalHostname: "localhost",
		HTTPPort:         "8080",
		SSHPort:          "2222",
		RootPassword:     "gitlabsmith123",
		DataVolumePath:   "/tmp/gitlab-smith/data",
		ConfigVolumePath: "/tmp/gitlab-smith/config",
		LogsVolumePath:   "/tmp/gitlab-smith/logs",
	}
}

// Deployer manages GitLab Docker deployments
type Deployer struct {
	config *DeploymentConfig
	ctx    context.Context
}

// New creates a new Deployer instance
func New(config *DeploymentConfig) *Deployer {
	if config == nil {
		config = DefaultConfig()
	}

	return &Deployer{
		config: config,
		ctx:    context.Background(),
	}
}

// Deploy deploys a new GitLab instance using Docker
func (d *Deployer) Deploy() error {
	// Check if Docker is available
	if err := d.checkDockerAvailability(); err != nil {
		return fmt.Errorf("docker check failed: %w", err)
	}

	// Stop and remove existing container if it exists
	if err := d.cleanup(); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	// Create data directories
	if err := d.createDataDirectories(); err != nil {
		return fmt.Errorf("failed to create data directories: %w", err)
	}

	// Run GitLab container
	if err := d.runContainer(); err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	// Wait for GitLab to be ready
	if err := d.waitForReadiness(); err != nil {
		return fmt.Errorf("gitlab failed to become ready: %w", err)
	}

	return nil
}

// checkDockerAvailability verifies Docker is installed and running
func (d *Deployer) checkDockerAvailability() error {
	cmd := exec.CommandContext(d.ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available or not running: %w", err)
	}
	return nil
}

// cleanup stops and removes existing GitLab container
func (d *Deployer) cleanup() error {
	// Stop container if running
	stopCmd := exec.CommandContext(d.ctx, "docker", "stop", d.config.ContainerName)
	_ = stopCmd.Run() // Ignore errors, container might not exist

	// Remove container if exists
	rmCmd := exec.CommandContext(d.ctx, "docker", "rm", d.config.ContainerName)
	_ = rmCmd.Run() // Ignore errors, container might not exist

	return nil
}

// createDataDirectories creates the necessary volume directories
func (d *Deployer) createDataDirectories() error {
	dirs := []string{
		d.config.DataVolumePath,
		d.config.ConfigVolumePath,
		d.config.LogsVolumePath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// runContainer starts the GitLab Docker container
func (d *Deployer) runContainer() error {
	args := []string{
		"run", "-d",
		"--hostname", d.config.ExternalHostname,
		"--name", d.config.ContainerName,
		"-p", fmt.Sprintf("%s:80", d.config.HTTPPort),
		"-p", fmt.Sprintf("%s:22", d.config.SSHPort),
		"--restart", "unless-stopped",
		"-v", fmt.Sprintf("%s:/var/opt/gitlab", d.config.DataVolumePath),
		"-v", fmt.Sprintf("%s:/etc/gitlab", d.config.ConfigVolumePath),
		"-v", fmt.Sprintf("%s:/var/log/gitlab", d.config.LogsVolumePath),
		"-e", fmt.Sprintf("GITLAB_ROOT_PASSWORD=%s", d.config.RootPassword),
		"-e", fmt.Sprintf("EXTERNAL_URL=http://%s:%s", d.config.ExternalHostname, d.config.HTTPPort),
		d.config.GitLabImage,
	}

	cmd := exec.CommandContext(d.ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run docker container: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// waitForReadiness waits for GitLab to be ready to accept requests
func (d *Deployer) waitForReadiness() error {
	maxRetries := 60 // 10 minutes with 10-second intervals
	retryInterval := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		if d.isReady() {
			return nil
		}

		fmt.Printf("Waiting for GitLab to be ready... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("gitlab did not become ready within timeout period")
}

// isReady checks if GitLab is ready by checking container health
func (d *Deployer) isReady() bool {
	// Check if container is running
	cmd := exec.CommandContext(d.ctx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", d.config.ContainerName), "--format", "{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	status := strings.TrimSpace(string(output))
	if !strings.Contains(status, "Up") {
		return false
	}

	// Check GitLab internal health status via docker exec
	healthCmd := exec.CommandContext(d.ctx, "docker", "exec", d.config.ContainerName, "gitlab-ctl", "status")
	if err := healthCmd.Run(); err != nil {
		return false
	}

	return true
}

// Destroy stops and removes the GitLab deployment
func (d *Deployer) Destroy() error {
	if err := d.cleanup(); err != nil {
		return fmt.Errorf("failed to destroy deployment: %w", err)
	}
	return nil
}

// GetStatus returns the current status of the GitLab deployment
func (d *Deployer) GetStatus() (*DeploymentStatus, error) {
	status := &DeploymentStatus{
		ContainerName: d.config.ContainerName,
		IsRunning:     false,
		URL:           fmt.Sprintf("http://%s:%s", d.config.ExternalHostname, d.config.HTTPPort),
	}

	// Check if container exists and is running
	cmd := exec.CommandContext(d.ctx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", d.config.ContainerName), "--format", "{{.Names}}\t{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return status, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 && parts[0] == d.config.ContainerName {
				status.IsRunning = strings.Contains(parts[1], "Up")
				status.ContainerStatus = parts[1]
				break
			}
		}
	}

	return status, nil
}

// GetLogs retrieves logs from the GitLab container
func (d *Deployer) GetLogs(w io.Writer, follow bool) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, d.config.ContainerName)

	cmd := exec.CommandContext(d.ctx, "docker", args...)
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd.Run()
}

// DeploymentStatus represents the current status of a GitLab deployment
type DeploymentStatus struct {
	ContainerName   string
	IsRunning       bool
	ContainerStatus string
	URL             string
}
