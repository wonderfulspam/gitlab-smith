package maintainability

import (
	"fmt"
	"strings"

	"github.com/wonderfulspam/gitlab-smith/pkg/analyzer/types"
	"github.com/wonderfulspam/gitlab-smith/pkg/parser"
)

func CheckDuplicatedCode(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	scriptSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		scriptKey := strings.Join(job.Script, "\n")
		if scriptKey != "" {
			scriptSets[scriptKey] = append(scriptSets[scriptKey], jobName)
		}
	}

	for _, jobNames := range scriptSets {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs",
				Message:    "Duplicated scripts in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider using extends or before_script to reduce duplication",
			})
		}
	}

	return issues
}

func CheckDuplicatedBeforeScripts(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	beforeScriptSets := make(map[string][]string)
	beforeScriptJobs := make(map[string][]string) // Map job name to before_script lines

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if len(job.BeforeScript) > 0 {
			scriptKey := strings.Join(job.BeforeScript, "\n")
			beforeScriptSets[scriptKey] = append(beforeScriptSets[scriptKey], jobName)
			beforeScriptJobs[jobName] = job.BeforeScript
		}
	}

	// Report exact duplicates
	for _, jobNames := range beforeScriptSets {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityHigh,
				Path:       "jobs.*.before_script",
				Message:    "Duplicate before_script blocks in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider consolidating duplicated before_script into default configuration or templates",
			})
		}
	}

	// Check for similar before_script blocks with high overlap
	processed := make(map[string]bool)
	for job1, script1 := range beforeScriptJobs {
		if processed[job1] {
			continue
		}
		similarJobs := []string{job1}
		for job2, script2 := range beforeScriptJobs {
			if job1 == job2 || processed[job2] {
				continue
			}
			// Calculate overlap between scripts
			overlap := calculateScriptOverlap(script1, script2)
			if overlap > 0.7 { // More than 70% overlap
				similarJobs = append(similarJobs, job2)
				processed[job2] = true
			}
		}
		processed[job1] = true

		if len(similarJobs) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs.*.before_script",
				Message:    "Similar before_script blocks with high overlap in jobs: " + strings.Join(similarJobs, ", "),
				Suggestion: "Consider extracting common commands to a shared template or default configuration",
			})
		}
	}

	return issues
}

// calculateScriptOverlap calculates the overlap percentage between two script blocks
func calculateScriptOverlap(script1, script2 []string) float64 {
	if len(script1) == 0 || len(script2) == 0 {
		return 0
	}

	// Count common lines
	set1 := make(map[string]bool)
	for _, line := range script1 {
		set1[strings.TrimSpace(line)] = true
	}

	commonCount := 0
	for _, line := range script2 {
		if set1[strings.TrimSpace(line)] {
			commonCount++
		}
	}

	// Calculate overlap as ratio of common lines to average length
	avgLen := float64(len(script1)+len(script2)) / 2.0
	return float64(commonCount) / avgLen
}

func CheckDuplicatedCacheConfig(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	cacheSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if job.Cache != nil {
			// Create a unique key for the cache configuration
			cacheKey := fmt.Sprintf("key:%s_paths:%s", job.Cache.Key, strings.Join(job.Cache.Paths, ","))
			cacheSets[cacheKey] = append(cacheSets[cacheKey], jobName)
		}
	}

	// Report duplicate cache configurations
	for _, jobNames := range cacheSets {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs.*.cache",
				Message:    "Duplicate cache configuration in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider consolidating duplicate cache configuration into default block or templates",
			})
		}
	}

	return issues
}

func CheckDuplicatedImageConfig(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	imageSets := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}
		if job.Image != "" {
			imageSets[job.Image] = append(imageSets[job.Image], jobName)
		}
	}

	// Report duplicate image configurations
	for image, jobNames := range imageSets {
		if len(jobNames) > 2 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityLow,
				Path:       "jobs.*.image",
				Message:    fmt.Sprintf("Duplicate image configuration '%s' in %d jobs: %s", image, len(jobNames), strings.Join(jobNames, ", ")),
				Suggestion: "Consider consolidating duplicate image configuration into default block",
			})
		}
	}

	return issues
}

func CheckDuplicatedSetup(config *parser.GitLabConfig) []types.Issue {
	var issues []types.Issue
	setupPatterns := make(map[string][]string)

	// Also check for overall duplication patterns across before_script and script
	overallSetupPatterns := make(map[string][]string)

	for jobName, job := range config.Jobs {
		// Skip template jobs (starting with .) from duplication analysis
		if strings.HasPrefix(jobName, ".") {
			continue
		}

		// Collect all setup-like commands from both before_script and script
		var allCommands []string
		allCommands = append(allCommands, job.BeforeScript...)
		allCommands = append(allCommands, job.Script...)

		// Check for common setup patterns
		for _, line := range allCommands {
			// Common package installation commands
			if strings.Contains(line, "npm ci") || strings.Contains(line, "npm install") ||
				strings.Contains(line, "pip install") || strings.Contains(line, "bundle install") ||
				strings.Contains(line, "composer install") || strings.Contains(line, "yarn install") {
				setupPatterns[line] = append(setupPatterns[line], jobName)
			}
			// System package installation
			if strings.Contains(line, "apt-get install") || strings.Contains(line, "yum install") ||
				strings.Contains(line, "apk add") {
				setupPatterns[line] = append(setupPatterns[line], jobName)
			}
			// Docker login and kubectl installation patterns
			if strings.Contains(line, "docker login") || strings.Contains(line, "kubectl") && strings.Contains(line, "curl") {
				setupPatterns[line] = append(setupPatterns[line], jobName)
			}
		}

		// Create a fingerprint of the job's overall setup configuration
		if len(job.BeforeScript) > 0 || (len(job.Script) > 0 && containsSetupCommands(job.Script)) {
			fingerprint := createSetupFingerprint(job)
			if fingerprint != "" {
				overallSetupPatterns[fingerprint] = append(overallSetupPatterns[fingerprint], jobName)
			}
		}
	}

	// Report duplicate setup patterns
	for pattern, jobNames := range setupPatterns {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs.*.script",
				Message:    fmt.Sprintf("Duplicate setup configuration '%s' in jobs: %s", pattern, strings.Join(jobNames, ", ")),
				Suggestion: "Consider moving setup commands to before_script or default configuration",
			})
		}
	}

	// Report jobs with similar overall setup configuration
	for _, jobNames := range overallSetupPatterns {
		if len(jobNames) > 1 {
			issues = append(issues, types.Issue{
				Type:       types.IssueTypeMaintainability,
				Severity:   types.SeverityMedium,
				Path:       "jobs",
				Message:    "Duplicate setup configuration in jobs: " + strings.Join(jobNames, ", "),
				Suggestion: "Consider using templates or default configuration to reduce duplication",
			})
		}
	}

	return issues
}

// containsSetupCommands checks if script contains setup-like commands
func containsSetupCommands(script []string) bool {
	for _, line := range script {
		if strings.Contains(line, "install") || strings.Contains(line, "apt-get") ||
			strings.Contains(line, "docker login") || strings.Contains(line, "curl") ||
			strings.Contains(line, "npm ci") || strings.Contains(line, "cache clean") {
			return true
		}
	}
	return false
}

// createSetupFingerprint creates a fingerprint of setup commands in a job
func createSetupFingerprint(job *parser.JobConfig) string {
	var setupCommands []string

	// Extract setup-related commands
	for _, line := range job.BeforeScript {
		if isSetupCommand(line) {
			// Normalize the command for comparison
			normalized := normalizeSetupCommand(line)
			if normalized != "" {
				setupCommands = append(setupCommands, normalized)
			}
		}
	}

	for _, line := range job.Script {
		if isSetupCommand(line) {
			normalized := normalizeSetupCommand(line)
			if normalized != "" {
				setupCommands = append(setupCommands, normalized)
			}
		}
	}

	if len(setupCommands) > 0 {
		return strings.Join(setupCommands, "|")
	}
	return ""
}

// isSetupCommand checks if a command is setup-related
func isSetupCommand(cmd string) bool {
	setupKeywords := []string{
		"apt-get", "yum", "apk", "install", "npm ci", "npm cache",
		"docker login", "kubectl", "curl", "wget", "pip install",
		"bundle install", "composer install", "yarn install",
		"--version", "echo", "sleep",
	}

	for _, keyword := range setupKeywords {
		if strings.Contains(cmd, keyword) {
			return true
		}
	}
	return false
}

// normalizeSetupCommand normalizes a setup command for comparison
func normalizeSetupCommand(cmd string) string {
	// Remove variable references and normalize whitespace
	cmd = strings.TrimSpace(cmd)

	// Extract the core command pattern
	if strings.Contains(cmd, "apt-get install") {
		return "apt-get-install"
	}
	if strings.Contains(cmd, "npm ci") {
		return "npm-ci"
	}
	if strings.Contains(cmd, "npm cache clean") {
		return "npm-cache-clean"
	}
	if strings.Contains(cmd, "docker login") {
		return "docker-login"
	}
	if strings.Contains(cmd, "kubectl") && strings.Contains(cmd, "curl") {
		return "kubectl-install"
	}
	if strings.Contains(cmd, "--version") {
		return "version-check"
	}

	return ""
}
