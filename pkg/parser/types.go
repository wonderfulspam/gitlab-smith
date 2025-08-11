package parser

// GitLabConfig represents a parsed GitLab CI configuration
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
