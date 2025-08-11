package renderer

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/emt/gitlab-smith/pkg/parser"
)

// VisualFormat represents supported visual diagram formats
type VisualFormat string

const (
	FormatDOT     VisualFormat = "dot"
	FormatMermaid VisualFormat = "mermaid"
)

// VisualRenderer handles generation of visual pipeline representations
type VisualRenderer struct{}

// NewVisualRenderer creates a new VisualRenderer instance
func NewVisualRenderer() *VisualRenderer {
	return &VisualRenderer{}
}

// RenderPipelineGraph generates a visual representation of pipeline structure
func (vr *VisualRenderer) RenderPipelineGraph(config *parser.GitLabConfig, format VisualFormat) (string, error) {
	switch format {
	case FormatDOT:
		return vr.generateDOTGraph(config), nil
	case FormatMermaid:
		return vr.generateMermaidGraph(config), nil
	default:
		return "", fmt.Errorf("unsupported visual format: %s", format)
	}
}

// RenderComparisonGraph generates a side-by-side visual comparison
func (vr *VisualRenderer) RenderComparisonGraph(oldConfig, newConfig *parser.GitLabConfig, comparison *PipelineComparison, format VisualFormat) (string, error) {
	switch format {
	case FormatDOT:
		return vr.generateComparisonDOTGraph(oldConfig, newConfig, comparison), nil
	case FormatMermaid:
		return vr.generateComparisonMermaidGraph(oldConfig, newConfig, comparison), nil
	default:
		return "", fmt.Errorf("unsupported visual format: %s", format)
	}
}

// generateDOTGraph creates a DOT graph representation of the pipeline
func (vr *VisualRenderer) generateDOTGraph(config *parser.GitLabConfig) string {
	var buf bytes.Buffer

	buf.WriteString("digraph pipeline {\n")
	buf.WriteString("  rankdir=TB;\n")
	buf.WriteString("  node [shape=box, style=rounded];\n")
	buf.WriteString("  edge [arrowhead=open];\n\n")

	// Group jobs by stages
	stageJobs := vr.groupJobsByStage(config)

	// Create subgraphs for each stage
	for i, stage := range config.Stages {
		jobs := stageJobs[stage]
		if len(jobs) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", i))
		buf.WriteString(fmt.Sprintf("    label=\"%s\";\n", stage))
		buf.WriteString("    style=filled;\n")
		buf.WriteString("    color=lightgrey;\n")

		for _, jobName := range jobs {
			job := config.Jobs[jobName]
			if job == nil {
				continue
			}

			nodeColor := vr.getJobNodeColor(job)
			buf.WriteString(fmt.Sprintf("    \"%s\" [fillcolor=%s, style=\"filled,rounded\"];\n", jobName, nodeColor))
		}

		buf.WriteString("  }\n\n")
	}

	// Add dependencies
	dependencyGraph := config.GetDependencyGraph()
	for jobName, deps := range dependencyGraph {
		for _, dep := range deps {
			buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", dep, jobName))
		}
	}

	buf.WriteString("}\n")
	return buf.String()
}

// generateMermaidGraph creates a Mermaid flowchart representation
func (vr *VisualRenderer) generateMermaidGraph(config *parser.GitLabConfig) string {
	var buf bytes.Buffer

	buf.WriteString("flowchart TD\n")

	// Group jobs by stages
	stageJobs := vr.groupJobsByStage(config)

	// Create stage subgraphs
	for i, stage := range config.Stages {
		jobs := stageJobs[stage]
		if len(jobs) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintf("  subgraph S%d[\"%s\"]\n", i, stage))

		for _, jobName := range jobs {
			job := config.Jobs[jobName]
			if job == nil {
				continue
			}

			nodeStyle := vr.getMermaidNodeStyle(job)
			buf.WriteString(fmt.Sprintf("    %s%s\n", vr.sanitizeMermaidID(jobName), nodeStyle))
		}

		buf.WriteString("  end\n\n")
	}

	// Add dependencies
	dependencyGraph := config.GetDependencyGraph()
	for jobName, deps := range dependencyGraph {
		for _, dep := range deps {
			buf.WriteString(fmt.Sprintf("  %s --> %s\n", vr.sanitizeMermaidID(dep), vr.sanitizeMermaidID(jobName)))
		}
	}

	// Add styling
	buf.WriteString("\n  classDef buildJob fill:#e1f5fe;\n")
	buf.WriteString("  classDef testJob fill:#f3e5f5;\n")
	buf.WriteString("  classDef deployJob fill:#e8f5e8;\n")
	buf.WriteString("  classDef defaultJob fill:#fff3e0;\n")

	return buf.String()
}

// generateComparisonDOTGraph creates a DOT graph showing before/after comparison
func (vr *VisualRenderer) generateComparisonDOTGraph(oldConfig, newConfig *parser.GitLabConfig, comparison *PipelineComparison) string {
	var buf bytes.Buffer

	buf.WriteString("digraph comparison {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box, style=rounded];\n")
	buf.WriteString("  edge [arrowhead=open];\n\n")

	buf.WriteString("  subgraph cluster_old {\n")
	buf.WriteString("    label=\"Before\";\n")
	buf.WriteString("    style=filled;\n")
	buf.WriteString("    color=lightcoral;\n")
	buf.WriteString(vr.generateSubgraphContent(oldConfig, "old_"))
	buf.WriteString("  }\n\n")

	buf.WriteString("  subgraph cluster_new {\n")
	buf.WriteString("    label=\"After\";\n")
	buf.WriteString("    style=filled;\n")
	buf.WriteString("    color=lightgreen;\n")
	buf.WriteString(vr.generateSubgraphContent(newConfig, "new_"))
	buf.WriteString("  }\n\n")

	// Add comparison highlighting
	for _, jobComp := range comparison.JobComparisons {
		if jobComp.OldJob != nil && jobComp.NewJob != nil {
			color := vr.getComparisonEdgeColor(jobComp.Status)
			buf.WriteString(fmt.Sprintf("  \"old_%s\" -> \"new_%s\" [style=dashed, color=%s, constraint=false];\n",
				jobComp.JobName, jobComp.JobName, color))
		}
	}

	buf.WriteString("}\n")
	return buf.String()
}

// generateComparisonMermaidGraph creates a Mermaid diagram showing before/after comparison
func (vr *VisualRenderer) generateComparisonMermaidGraph(oldConfig, newConfig *parser.GitLabConfig, comparison *PipelineComparison) string {
	var buf bytes.Buffer

	buf.WriteString("flowchart LR\n")
	buf.WriteString("  subgraph B[\"Before\"]\n")
	buf.WriteString(vr.generateMermaidSubgraph(oldConfig, "b"))
	buf.WriteString("  end\n\n")

	buf.WriteString("  subgraph A[\"After\"]\n")
	buf.WriteString(vr.generateMermaidSubgraph(newConfig, "a"))
	buf.WriteString("  end\n\n")

	// Add comparison connections
	for _, jobComp := range comparison.JobComparisons {
		if jobComp.OldJob != nil && jobComp.NewJob != nil {
			style := vr.getComparisonMermaidStyle(jobComp.Status)
			buf.WriteString(fmt.Sprintf("  b%s %s a%s\n",
				vr.sanitizeMermaidID(jobComp.JobName), style, vr.sanitizeMermaidID(jobComp.JobName)))
		}
	}

	// Add styling for comparison types
	buf.WriteString("\n  classDef improved stroke:#4caf50,stroke-width:3px;\n")
	buf.WriteString("  classDef degraded stroke:#f44336,stroke-width:3px;\n")
	buf.WriteString("  classDef identical stroke:#2196f3,stroke-width:2px;\n")
	buf.WriteString("  classDef added fill:#c8e6c9;\n")
	buf.WriteString("  classDef removed fill:#ffcdd2;\n")

	return buf.String()
}

// Helper methods

func (vr *VisualRenderer) groupJobsByStage(config *parser.GitLabConfig) map[string][]string {
	stageJobs := make(map[string][]string)

	for jobName, job := range config.Jobs {
		if job == nil {
			continue
		}

		// Skip template jobs (starting with .)
		if strings.HasPrefix(jobName, ".") {
			continue
		}

		stage := job.Stage
		if stage == "" {
			stage = "test" // Default stage
		}

		stageJobs[stage] = append(stageJobs[stage], jobName)
	}

	// Sort jobs within each stage for consistent output
	for stage := range stageJobs {
		sort.Strings(stageJobs[stage])
	}

	return stageJobs
}

func (vr *VisualRenderer) getJobNodeColor(job *parser.JobConfig) string {
	stage := strings.ToLower(job.Stage)

	switch {
	case strings.Contains(stage, "build"):
		return "lightblue"
	case strings.Contains(stage, "test"):
		return "lightpink"
	case strings.Contains(stage, "deploy"):
		return "lightgreen"
	default:
		return "lightyellow"
	}
}

func (vr *VisualRenderer) getMermaidNodeStyle(job *parser.JobConfig) string {
	stage := strings.ToLower(job.Stage)
	jobName := job.Stage // Use stage as the display name for simplicity

	var class string
	switch {
	case strings.Contains(stage, "build"):
		class = "buildJob"
	case strings.Contains(stage, "test"):
		class = "testJob"
	case strings.Contains(stage, "deploy"):
		class = "deployJob"
	default:
		class = "defaultJob"
	}

	return fmt.Sprintf("[\"%s\"]:::%s", jobName, class)
}

func (vr *VisualRenderer) sanitizeMermaidID(id string) string {
	// Replace characters that aren't valid in Mermaid IDs
	sanitized := strings.ReplaceAll(id, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}

func (vr *VisualRenderer) generateSubgraphContent(config *parser.GitLabConfig, prefix string) string {
	var buf bytes.Buffer

	stageJobs := vr.groupJobsByStage(config)

	for _, stage := range config.Stages {
		jobs := stageJobs[stage]
		if len(jobs) == 0 {
			continue
		}

		for _, jobName := range jobs {
			job := config.Jobs[jobName]
			if job == nil {
				continue
			}

			nodeColor := vr.getJobNodeColor(job)
			buf.WriteString(fmt.Sprintf("    \"%s%s\" [fillcolor=%s, style=\"filled,rounded\"];\n", prefix, jobName, nodeColor))
		}
	}

	// Add dependencies within this subgraph
	dependencyGraph := config.GetDependencyGraph()
	for jobName, deps := range dependencyGraph {
		for _, dep := range deps {
			buf.WriteString(fmt.Sprintf("    \"%s%s\" -> \"%s%s\";\n", prefix, dep, prefix, jobName))
		}
	}

	return buf.String()
}

func (vr *VisualRenderer) generateMermaidSubgraph(config *parser.GitLabConfig, prefix string) string {
	var buf bytes.Buffer

	stageJobs := vr.groupJobsByStage(config)

	for _, stage := range config.Stages {
		jobs := stageJobs[stage]
		if len(jobs) == 0 {
			continue
		}

		for _, jobName := range jobs {
			job := config.Jobs[jobName]
			if job == nil {
				continue
			}

			buf.WriteString(fmt.Sprintf("    %s%s[\"%s\"]\n", prefix, vr.sanitizeMermaidID(jobName), jobName))
		}
	}

	// Add dependencies within this subgraph
	dependencyGraph := config.GetDependencyGraph()
	for jobName, deps := range dependencyGraph {
		for _, dep := range deps {
			buf.WriteString(fmt.Sprintf("    %s%s --> %s%s\n",
				prefix, vr.sanitizeMermaidID(dep), prefix, vr.sanitizeMermaidID(jobName)))
		}
	}

	return buf.String()
}

func (vr *VisualRenderer) getComparisonEdgeColor(status CompareStatus) string {
	switch status {
	case StatusImproved:
		return "green"
	case StatusDegraded:
		return "red"
	case StatusIdentical:
		return "blue"
	case StatusRestructured:
		return "orange"
	default:
		return "gray"
	}
}

func (vr *VisualRenderer) getComparisonMermaidStyle(status CompareStatus) string {
	switch status {
	case StatusImproved:
		return "-.->|improved|"
	case StatusDegraded:
		return "-.->|degraded|"
	case StatusIdentical:
		return "-.->|identical|"
	case StatusRestructured:
		return "-.->|changed|"
	default:
		return "-..-"
	}
}
