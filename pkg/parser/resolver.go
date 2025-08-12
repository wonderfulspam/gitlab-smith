package parser

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IncludeResolver handles resolution of different include types
type IncludeResolver struct {
	httpClient   *http.Client
	cache        map[string][]byte
	gitlabAPIURL string
	gitlabToken  string
}

// NewIncludeResolver creates a new include resolver with optional GitLab API configuration
func NewIncludeResolver(gitlabAPIURL, gitlabToken string) *IncludeResolver {
	return &IncludeResolver{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:        make(map[string][]byte),
		gitlabAPIURL: gitlabAPIURL,
		gitlabToken:  gitlabToken,
	}
}

// ResolveIncludes resolves and merges include files into the configuration
func ResolveIncludes(config *GitLabConfig, baseDir string) error {
	resolver := NewIncludeResolver("", "")
	return ResolveIncludesWithResolver(config, baseDir, resolver)
}

// ResolveIncludesWithResolver resolves includes using a custom resolver
func ResolveIncludesWithResolver(config *GitLabConfig, baseDir string, resolver *IncludeResolver) error {
	for _, include := range config.Include {
		var data []byte
		var err error

		if include.Local != "" {
			// Resolve local includes
			includePath := filepath.Join(baseDir, include.Local)
			data, err = resolver.resolveLocalInclude(includePath)
		} else if include.Remote != "" {
			// Resolve remote includes
			data, err = resolver.resolveRemoteInclude(include.Remote)
		} else if include.Template != "" {
			// Resolve GitLab template includes
			data, err = resolver.resolveTemplateInclude(include.Template)
		} else if include.Project != "" && len(include.File) > 0 {
			// Resolve project includes
			data, err = resolver.resolveProjectInclude(include.Project, include.File[0], include.Ref)
		}

		if err != nil {
			// Continue processing other includes even if one fails
			// This matches GitLab's behavior of gracefully handling missing includes
			continue
		}

		if data != nil {
			if err := resolver.mergeIncludedData(config, data, baseDir); err != nil {
				continue
			}
		}
	}
	return nil
}

// resolveLocalInclude reads a local file
func (r *IncludeResolver) resolveLocalInclude(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// resolveRemoteInclude fetches a remote file via HTTP/HTTPS
func (r *IncludeResolver) resolveRemoteInclude(url string) ([]byte, error) {
	// Check cache first
	if cached, exists := r.cache[url]; exists {
		return cached, nil
	}

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote include %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote include %s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote include %s: %w", url, err)
	}

	// Cache the result
	r.cache[url] = data
	return data, nil
}

// resolveTemplateInclude resolves GitLab-provided templates
func (r *IncludeResolver) resolveTemplateInclude(template string) ([]byte, error) {
	// GitLab templates are hosted on GitLab.com
	baseURL := "https://gitlab.com/gitlab-org/gitlab/-/raw/master/lib/gitlab/ci/templates/"

	// Ensure template has .yml extension
	if !strings.HasSuffix(template, ".yml") && !strings.HasSuffix(template, ".yaml") {
		template += ".yml"
	}

	url := baseURL + template
	return r.resolveRemoteInclude(url)
}

// resolveProjectInclude resolves includes from other GitLab projects
func (r *IncludeResolver) resolveProjectInclude(project, file, ref string) ([]byte, error) {
	if r.gitlabAPIURL == "" {
		return nil, fmt.Errorf("GitLab API URL not configured for project includes")
	}

	// Default ref if not specified
	if ref == "" {
		ref = "HEAD"
	}

	// Build GitLab API URL for file content
	// Format: /projects/:id/repository/files/:file_path/raw?ref=:ref
	url := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s",
		strings.TrimSuffix(r.gitlabAPIURL, "/"),
		strings.Replace(project, "/", "%2F", -1), // URL encode project path
		strings.Replace(file, "/", "%2F", -1),    // URL encode file path
		ref)

	// Check cache first
	cacheKey := fmt.Sprintf("project:%s:%s:%s", project, file, ref)
	if cached, exists := r.cache[cacheKey]; exists {
		return cached, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for project include: %w", err)
	}

	if r.gitlabToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.gitlabToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project include %s/%s: %w", project, file, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("project include %s/%s returned status %d", project, file, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read project include %s/%s: %w", project, file, err)
	}

	// Cache the result
	r.cache[cacheKey] = data
	return data, nil
}

// mergeIncludedData merges included YAML data into the configuration
func (r *IncludeResolver) mergeIncludedData(config *GitLabConfig, data []byte, baseDir string) error {
	includedConfig, err := Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse included data: %w", err)
	}

	// Merge included configuration into main config
	// Jobs from includes are added (later includes can override earlier ones)
	for jobName, job := range includedConfig.Jobs {
		if config.Jobs == nil {
			config.Jobs = make(map[string]*JobConfig)
		}
		config.Jobs[jobName] = job
	}

	// Merge variables (included variables are overridden by main file)
	if includedConfig.Variables != nil && config.Variables == nil {
		config.Variables = includedConfig.Variables
	}

	// Stages are typically only defined in the main file, but merge if needed
	if len(config.Stages) == 0 && len(includedConfig.Stages) > 0 {
		config.Stages = includedConfig.Stages
	}

	// Default job config
	if config.Default == nil && includedConfig.Default != nil {
		config.Default = includedConfig.Default
	}

	// Recursively process includes from the included file
	if len(includedConfig.Include) > 0 {
		if err := ResolveIncludesWithResolver(includedConfig, baseDir, r); err != nil {
			return err
		}
		// Then merge any additional jobs found
		for jobName, job := range includedConfig.Jobs {
			if _, exists := config.Jobs[jobName]; !exists {
				config.Jobs[jobName] = job
			}
		}
	}

	return nil
}
