package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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
