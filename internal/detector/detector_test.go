package detector

import (
	"path/filepath"
	"testing"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestDetectorOnActualProjects(t *testing.T) {
	d := New()

	tests := []struct {
		relPath  string
		expected model.ProjectID
	}{
		{"../../../libft", model.ProjectLibft},
		{"../../../ft-printf", model.ProjectFtPrintf},
		{"../../../get-next-line", model.ProjectGetNextLine},
		{"../../../push-swap", model.ProjectPushSwap},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.relPath)
			if err != nil {
				t.Fatalf("Failed to get abs path: %v", err)
			}
			res, err := d.Detect(absPath)
			if err != nil {
				t.Fatalf("Detection failed with error: %v", err)
			}
			if res.Project.ID != tt.expected {
				t.Errorf("Expected %s, got %s (confidence: %.2f)", tt.expected, res.Project.ID, res.Confidence)
			}
			t.Logf("Project %s detected with confidence %.2f (Bonus: %v, HasMakefile: %v)",
				res.Project.Name, res.Confidence, res.HasBonus, res.HasMakefile)
		})
	}
}
