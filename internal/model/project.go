package model

// ProjectID represents a unique identifier for a 42 project.
type ProjectID string

const (
	ProjectLibft       ProjectID = "libft"
	ProjectFtPrintf    ProjectID = "ft_printf"
	ProjectGetNextLine ProjectID = "get_next_line"
	ProjectPushSwap    ProjectID = "push_swap"
	ProjectUnknown     ProjectID = "unknown"
)

// ProjectMetadata describes a 42 project curriculum details.
type ProjectMetadata struct {
	ID             ProjectID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SubjectVersion string    `json:"subject_version"`
	HasMakefile    bool      `json:"has_makefile"`
	BinaryName     string    `json:"binary_name,omitempty"`
	LibraryName    string    `json:"library_name,omitempty"`
	RequiredFiles  []string  `json:"required_files"`
	BonusFiles     []string  `json:"bonus_files,omitempty"`
}

// DetectionResult represents the outcome of detecting a project in a directory.
type DetectionResult struct {
	Project      ProjectMetadata `json:"project"`
	Confidence   float64         `json:"confidence"` // 0.0 to 1.0
	RootDir      string          `json:"root_dir"`
	FoundFiles   []string        `json:"found_files"`
	MissingFiles []string        `json:"missing_files"`
	HasBonus     bool            `json:"has_bonus"`
	HasMakefile  bool            `json:"has_makefile"`
}
