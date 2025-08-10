package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type GitLabConfig struct {
	Stages    []string               `yaml:"stages" json:"stages,omitempty"`
	Variables map[string]interface{} `yaml:"variables" json:"variables,omitempty"`
	Include   []Include              `yaml:"include" json:"include,omitempty"`
	Default   *JobConfig             `yaml:"default" json:"default,omitempty"`
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
	Extends       []string               `yaml:"extends,omitempty" json:"extends,omitempty"`
}

type Cache struct {
	Key       string   `yaml:"key,omitempty" json:"key,omitempty"`
	Paths     []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Policy    string   `yaml:"policy,omitempty" json:"policy,omitempty"`
	Untracked bool     `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	When      string   `yaml:"when,omitempty" json:"when,omitempty"`
}

type Artifacts struct {
	Paths     []string          `yaml:"paths,omitempty" json:"paths,omitempty"`
	Name      string            `yaml:"name,omitempty" json:"name,omitempty"`
	Untracked bool              `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	When      string            `yaml:"when,omitempty" json:"when,omitempty"`
	ExpireIn  string            `yaml:"expire_in,omitempty" json:"expire_in,omitempty"`
	Reports   map[string]string `yaml:"reports,omitempty" json:"reports,omitempty"`
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

func Parse(data []byte) (*GitLabConfig, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling YAML: %w", err)
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
		default:
			if !isReservedKeyword(key) && key[0] != '.' && isJobDefinition(value) {
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
