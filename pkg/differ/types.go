package differ

type DiffType string

const (
	DiffTypeAdded    DiffType = "added"
	DiffTypeRemoved  DiffType = "removed"
	DiffTypeModified DiffType = "modified"
	DiffTypeRenamed  DiffType = "renamed"
)

type ConfigDiff struct {
	Type        DiffType    `json:"type"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Behavioral  bool        `json:"behavioral"` // Whether this change affects pipeline behavior
}

type DiffResult struct {
	Semantic        []ConfigDiff `json:"semantic"`
	Dependencies    []ConfigDiff `json:"dependencies"`
	Performance     []ConfigDiff `json:"performance"`
	Improvements    []ConfigDiff `json:"improvements"` // Detected refactoring improvements
	HasChanges      bool         `json:"has_changes"`
	Summary         string       `json:"summary"`
	ImprovementTags []string     `json:"improvement_tags"` // Tags like "duplication", "consolidation", "templates"
}
