package parser

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type GitLabConfig struct {
	Stages    []string               `yaml:"stages" json:"stages,omitempty"`
	Variables map[string]interface{} `yaml:"variables" json:"variables,omitempty"`
	Include   []Include              `yaml:"include" json:"include,omitempty"`
	Default   *JobConfig             `yaml:"default" json:"default,omitempty"`
	Workflow  *Workflow              `yaml:"workflow" json:"workflow,omitempty"`
	Jobs      map[string]*JobConfig  `json:"jobs,omitempty"`
	RawData   map[string]interface{} `json:"-"`
}

type Include struct {
	Local    string   `yaml:"local,omitempty" json:"local,omitempty"`
	File     []string `yaml:"file,omitempty" json:"file,omitempty"`
	Template string   `yaml:"template,omitempty" json:"template,omitempty"`
	Remote   string   `yaml:"remote,omitempty" json:"remote,omitempty"`
	Project  string   `yaml:"project,omitempty" json:"project,omitempty"`
	Ref      string   `yaml:"ref,omitempty" json:"ref,omitempty"`
}

type JobConfig struct {
	Stage         string                 `yaml:"stage,omitempty" json:"stage,omitempty"`
	Script        []string               `yaml:"script,omitempty" json:"script,omitempty"`
	BeforeScript  []string               `yaml:"before_script,omitempty" json:"before_script,omitempty"`
	AfterScript   []string               `yaml:"after_script,omitempty" json:"after_script,omitempty"`
	Image         string                 `yaml:"image,omitempty" json:"image,omitempty"`
	Services      []string               `yaml:"services,omitempty" json:"services,omitempty"`
	Variables     map[string]interface{} `yaml:"variables,omitempty" json:"variables,omitempty"`
	Cache         *Cache                 `yaml:"cache,omitempty" json:"cache,omitempty"`
	Artifacts     *Artifacts             `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Dependencies  []string               `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Needs         interface{}            `yaml:"needs,omitempty" json:"needs,omitempty"`
	Tags          []string               `yaml:"tags,omitempty" json:"tags,omitempty"`
	AllowFailure  bool                   `yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	When          string                 `yaml:"when,omitempty" json:"when,omitempty"`
	Only          interface{}            `yaml:"only,omitempty" json:"only,omitempty"`
	Except        interface{}            `yaml:"except,omitempty" json:"except,omitempty"`
	Rules         []Rule                 `yaml:"rules,omitempty" json:"rules,omitempty"`
	Retry         *Retry                 `yaml:"retry,omitempty" json:"retry,omitempty"`
	Timeout       string                 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Parallel      int                    `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	ResourceGroup string                 `yaml:"resource_group,omitempty" json:"resource_group,omitempty"`
	Environment   *Environment           `yaml:"environment,omitempty" json:"environment,omitempty"`
	Coverage      string                 `yaml:"coverage,omitempty" json:"coverage,omitempty"`
	Extends       interface{}            `yaml:"extends,omitempty" json:"extends,omitempty"`
}

type Cache struct {
	Key       string   `yaml:"key,omitempty" json:"key,omitempty"`
	Paths     []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Policy    string   `yaml:"policy,omitempty" json:"policy,omitempty"`
	Untracked bool     `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	When      string   `yaml:"when,omitempty" json:"when,omitempty"`
}

type Artifacts struct {
	Paths     []string               `yaml:"paths,omitempty" json:"paths,omitempty"`
	Name      string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Untracked bool                   `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	When      string                 `yaml:"when,omitempty" json:"when,omitempty"`
	ExpireIn  string                 `yaml:"expire_in,omitempty" json:"expire_in,omitempty"`
	Reports   map[string]interface{} `yaml:"reports,omitempty" json:"reports,omitempty"`
}

type Need struct {
	Job      string `yaml:"job,omitempty" json:"job,omitempty"`
	Ref      string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Pipeline string `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Optional bool   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type OnlyExcept struct {
	Refs       []string               `yaml:"refs,omitempty" json:"refs,omitempty"`
	Variables  []string               `yaml:"variables,omitempty" json:"variables,omitempty"`
	Changes    []string               `yaml:"changes,omitempty" json:"changes,omitempty"`
	Kubernetes string                 `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`
	Raw        map[string]interface{} `yaml:",inline" json:"-"`
}

type Rule struct {
	If           string                 `yaml:"if,omitempty" json:"if,omitempty"`
	Changes      []string               `yaml:"changes,omitempty" json:"changes,omitempty"`
	Exists       []string               `yaml:"exists,omitempty" json:"exists,omitempty"`
	Variables    map[string]interface{} `yaml:"variables,omitempty" json:"variables,omitempty"`
	When         string                 `yaml:"when,omitempty" json:"when,omitempty"`
	AllowFailure bool                   `yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	StartIn      string                 `yaml:"start_in,omitempty" json:"start_in,omitempty"`
}

type Retry struct {
	Max  int    `yaml:"max,omitempty" json:"max,omitempty"`
	When string `yaml:"when,omitempty" json:"when,omitempty"`
}

type Environment struct {
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	URL        string `yaml:"url,omitempty" json:"url,omitempty"`
	OnStop     string `yaml:"on_stop,omitempty" json:"on_stop,omitempty"`
	Action     string `yaml:"action,omitempty" json:"action,omitempty"`
	AutoStopIn string `yaml:"auto_stop_in,omitempty" json:"auto_stop_in,omitempty"`
	Deployment string `yaml:"deployment,omitempty" json:"deployment,omitempty"`
}

type Workflow struct {
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty"`
}

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

func Parse(data []byte) (*GitLabConfig, error) {
	// First parse with anchor/alias resolution
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing YAML structure: %w", err)
	}

	// Resolve anchors and aliases
	resolvedData, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("resolving YAML anchors: %w", err)
	}

	// Parse the resolved YAML into our structure
	var raw map[string]interface{}
	if err := yaml.Unmarshal(resolvedData, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling resolved YAML: %w", err)
	}

	config := &GitLabConfig{
		Jobs:    make(map[string]*JobConfig),
		RawData: raw,
	}

	for key, value := range raw {
		switch key {
		case "stages":
			if stages, ok := value.([]interface{}); ok {
				for _, s := range stages {
					if str, ok := s.(string); ok {
						config.Stages = append(config.Stages, str)
					}
				}
			}
		case "variables":
			if vars, ok := value.(map[string]interface{}); ok {
				config.Variables = vars
			}
		case "include":
			parseInclude(value, config)
		case "default":
			jobBytes, _ := yaml.Marshal(value)
			var defaultJob JobConfig
			if err := yaml.Unmarshal(jobBytes, &defaultJob); err == nil {
				config.Default = &defaultJob
			}
		case "workflow":
			workflowBytes, _ := yaml.Marshal(value)
			var workflow Workflow
			if err := yaml.Unmarshal(workflowBytes, &workflow); err == nil {
				config.Workflow = &workflow
			}
		default:
			if !isReservedKeyword(key) && isJobDefinition(value) {
				jobBytes, _ := yaml.Marshal(value)
				var job JobConfig
				if err := yaml.Unmarshal(jobBytes, &job); err == nil {
					config.Jobs[key] = &job
				}
			}
		}
	}

	return config, nil
}

// GetExtends returns the extends field as a slice of strings, handling both string and []string cases
func (j *JobConfig) GetExtends() []string {
	if j.Extends == nil {
		return nil
	}

	switch v := j.Extends.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		var extends []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				extends = append(extends, str)
			}
		}
		return extends
	default:
		return nil
	}
}

func parseInclude(value interface{}, config *GitLabConfig) {
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if includeMap, ok := item.(map[string]interface{}); ok {
				var include Include
				includeBytes, _ := yaml.Marshal(includeMap)
				if err := yaml.Unmarshal(includeBytes, &include); err == nil {
					config.Include = append(config.Include, include)
				}
			}
		}
	case map[string]interface{}:
		var include Include
		includeBytes, _ := yaml.Marshal(v)
		if err := yaml.Unmarshal(includeBytes, &include); err == nil {
			config.Include = append(config.Include, include)
		}
	case string:
		config.Include = append(config.Include, Include{Local: v})
	}
}

func isReservedKeyword(key string) bool {
	reserved := []string{
		"stages", "variables", "include", "default", "before_script", "after_script",
		"image", "services", "cache", "artifacts", "workflow",
	}
	for _, r := range reserved {
		if key == r {
			return true
		}
	}
	return false
}

func isJobDefinition(value interface{}) bool {
	if valueMap, ok := value.(map[string]interface{}); ok {
		for key := range valueMap {
			if key == "script" || key == "stage" || key == "image" ||
				key == "before_script" || key == "after_script" ||
				key == "needs" || key == "dependencies" || key == "services" ||
				key == "environment" || key == "only" || key == "except" ||
				key == "rules" || key == "when" || key == "artifacts" ||
				key == "cache" || key == "variables" || key == "tags" ||
				key == "allow_failure" || key == "retry" || key == "coverage" ||
				key == "timeout" || key == "parallel" || key == "extends" {
				return true
			}
		}
	}
	return false
}

// ParseFile parses a GitLab CI file and resolves its includes
func ParseFile(filePath string) (*GitLabConfig, error) {
	return ParseFileWithResolver(filePath, NewIncludeResolver("", ""))
}

// ParseFileWithResolver parses a GitLab CI file using a custom resolver
func ParseFileWithResolver(filePath string, resolver *IncludeResolver) (*GitLabConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	config, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Resolve includes relative to the file's directory
	baseDir := filepath.Dir(filePath)
	if err := ResolveIncludesWithResolver(config, baseDir, resolver); err != nil {
		return nil, fmt.Errorf("failed to resolve includes: %w", err)
	}

	return config, nil
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

// mergeIncludedFile reads and merges an included YAML file into the configuration
func mergeIncludedFile(config *GitLabConfig, includePath string) error {
	data, err := os.ReadFile(includePath)
	if err != nil {
		// File not found is not a fatal error in GitLab CI
		return nil
	}

	includedConfig, err := Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse included file %s: %w", includePath, err)
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
		includeDir := filepath.Dir(includePath)
		// First resolve the includes from the included file into the included config
		if err := ResolveIncludes(includedConfig, includeDir); err != nil {
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

func (c *GitLabConfig) GetDependencyGraph() map[string][]string {
	graph := make(map[string][]string)

	for jobName, job := range c.Jobs {
		deps := []string{}

		if len(job.Dependencies) > 0 {
			deps = append(deps, job.Dependencies...)
		}

		// Handle needs field which can be string array or object array
		if job.Needs != nil {
			switch needs := job.Needs.(type) {
			case []interface{}:
				for _, need := range needs {
					switch n := need.(type) {
					case string:
						deps = append(deps, n)
					case map[string]interface{}:
						if jobName, ok := n["job"].(string); ok {
							deps = append(deps, jobName)
						}
					}
				}
			case []string:
				deps = append(deps, needs...)
			}
		}

		graph[jobName] = deps
	}

	return graph
}

// SimulateMainBranchPipeline simulates which jobs would run on main branch
func (c *GitLabConfig) SimulateMainBranchPipeline() map[string]bool {
	context := DefaultPipelineContext()
	return c.SimulatePipeline(context)
}

// SimulateMergeRequestPipeline simulates which jobs would run in a merge request
func (c *GitLabConfig) SimulateMergeRequestPipeline(sourceBranch string) map[string]bool {
	context := MergeRequestPipelineContext(sourceBranch)
	return c.SimulatePipeline(context)
}

// SimulatePipeline simulates which jobs would run in the given pipeline context
func (c *GitLabConfig) SimulatePipeline(context *PipelineContext) map[string]bool {
	result := make(map[string]bool)

	// First check if pipeline should be created at all
	evaluator := NewWorkflowEvaluator(c, context)
	if !evaluator.ShouldCreatePipeline() {
		// No jobs run if pipeline is not created
		return result
	}

	// Evaluate each job's rules to see if it should run
	for jobName, job := range c.Jobs {
		result[jobName] = c.shouldJobRun(job, context)
	}

	return result
}

// shouldJobRun evaluates if a job should run in the given context
func (c *GitLabConfig) shouldJobRun(job *JobConfig, context *PipelineContext) bool {
	// If job has rules, evaluate them
	if len(job.Rules) > 0 {
		return c.evaluateJobRules(job, context)
	}

	// If job has only/except, evaluate them (legacy)
	if job.Only != nil || job.Except != nil {
		return c.evaluateOnlyExcept(job, context)
	}

	// Default behavior: job runs
	return true
}

// evaluateJobRules evaluates job rules to determine if job should run
func (c *GitLabConfig) evaluateJobRules(job *JobConfig, context *PipelineContext) bool {
	for _, rule := range job.Rules {
		if c.ruleMatches(&rule, context) {
			switch rule.When {
			case "never":
				return false
			case "always", "":
				return true
			case "on_success", "on_failure", "manual", "delayed":
				// These depend on previous job status, assume true for simulation
				return true
			}
		}
	}

	// No rule matched, default behavior is job doesn't run
	return false
}

// ruleMatches checks if a rule matches the current context (simplified)
func (c *GitLabConfig) ruleMatches(rule *Rule, context *PipelineContext) bool {
	// If no conditions, rule matches
	if rule.If == "" && len(rule.Changes) == 0 && len(rule.Exists) == 0 {
		return true
	}

	// Simple if condition evaluation
	if rule.If != "" {
		return c.evaluateSimpleIfCondition(rule.If, context)
	}

	// For changes/exists, we can't evaluate without file system, assume true
	return len(rule.Changes) == 0 && len(rule.Exists) == 0
}

// evaluateSimpleIfCondition provides basic evaluation of if conditions
func (c *GitLabConfig) evaluateSimpleIfCondition(condition string, context *PipelineContext) bool {
	// This is a simplified version - in practice GitLab has complex expression evaluation
	condition = strings.TrimSpace(condition)

	// Common patterns
	if strings.Contains(condition, "$CI_PIPELINE_SOURCE == \"push\"") {
		return context.Event == "push"
	}
	if strings.Contains(condition, "$CI_PIPELINE_SOURCE == \"merge_request_event\"") {
		return context.Event == "merge_request_event"
	}
	if strings.Contains(condition, "$CI_COMMIT_BRANCH == \"main\"") ||
		strings.Contains(condition, "$CI_COMMIT_BRANCH == \"master\"") {
		return context.IsMainBranch
	}
	if strings.Contains(condition, "$CI_MERGE_REQUEST_ID") {
		return context.IsMR
	}

	// Default to true for unknown conditions
	return true
}

// evaluateOnlyExcept evaluates legacy only/except directives
func (c *GitLabConfig) evaluateOnlyExcept(job *JobConfig, context *PipelineContext) bool {
	// This is a simplified implementation of only/except logic
	// In practice, GitLab has complex matching rules for refs, variables, etc.

	// If only is specified, job runs only if conditions match
	if job.Only != nil {
		return c.matchesOnlyExcept(job.Only, context, true)
	}

	// If except is specified, job runs unless conditions match
	if job.Except != nil {
		return !c.matchesOnlyExcept(job.Except, context, false)
	}

	return true
}

// matchesOnlyExcept checks if only/except conditions match
func (c *GitLabConfig) matchesOnlyExcept(condition interface{}, context *PipelineContext, isOnly bool) bool {
	switch v := condition.(type) {
	case []interface{}:
		// Array of conditions
		for _, item := range v {
			if str, ok := item.(string); ok {
				if c.matchesSingleCondition(str, context) {
					return true
				}
			}
		}
	case []string:
		for _, str := range v {
			if c.matchesSingleCondition(str, context) {
				return true
			}
		}
	case string:
		return c.matchesSingleCondition(v, context)
	}

	return false
}

// matchesSingleCondition checks if a single condition string matches
func (c *GitLabConfig) matchesSingleCondition(condition string, context *PipelineContext) bool {
	switch condition {
	case "master", "main":
		return context.IsMainBranch
	case "merge_requests":
		return context.IsMR
	case "pushes":
		return context.Event == "push"
	default:
		// Could be a branch name or pattern
		return condition == context.Branch
	}
}
