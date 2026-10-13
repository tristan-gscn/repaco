package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestDetectorMockProjects(t *testing.T) {
	d := New()
	tmpDir := t.TempDir()

	// 1. Mock libft
	libftDir := filepath.Join(tmpDir, "libft")
	_ = os.MkdirAll(libftDir, 0755)
	_ = os.WriteFile(filepath.Join(libftDir, "libft.h"), []byte("#ifndef LIBFT_H\n#define LIBFT_H\n#endif"), 0644)
	_ = os.WriteFile(filepath.Join(libftDir, "Makefile"), []byte("all:\n"), 0644)

	res, err := d.Detect(libftDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if res.Project.ID != model.ProjectLibft {
		t.Errorf("Expected ProjectLibft, got %s", res.Project.ID)
	}

	// 2. Mock ft_printf
	printfDir := filepath.Join(tmpDir, "ft_printf")
	_ = os.MkdirAll(printfDir, 0755)
	_ = os.WriteFile(filepath.Join(printfDir, "ft_printf.h"), []byte("#ifndef FT_PRINTF_H\n#endif"), 0644)
	_ = os.WriteFile(filepath.Join(printfDir, "ft_printf.c"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(printfDir, "Makefile"), []byte("all:\n"), 0644)

	res, err = d.Detect(printfDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if res.Project.ID != model.ProjectFtPrintf {
		t.Errorf("Expected ProjectFtPrintf, got %s", res.Project.ID)
	}

	// 3. Mock get_next_line
	gnlDir := filepath.Join(tmpDir, "get_next_line")
	_ = os.MkdirAll(gnlDir, 0755)
	_ = os.WriteFile(filepath.Join(gnlDir, "get_next_line.h"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(gnlDir, "get_next_line.c"), []byte(""), 0644)

	res, err = d.Detect(gnlDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if res.Project.ID != model.ProjectGetNextLine {
		t.Errorf("Expected ProjectGetNextLine, got %s", res.Project.ID)
	}

	// 4. Mock push_swap
	psDir := filepath.Join(tmpDir, "push_swap")
	_ = os.MkdirAll(psDir, 0755)
	_ = os.WriteFile(filepath.Join(psDir, "push_swap.h"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(psDir, "push_swap.c"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(psDir, "Makefile"), []byte("all:\n"), 0644)

	res, err = d.Detect(psDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if res.Project.ID != model.ProjectPushSwap {
		t.Errorf("Expected ProjectPushSwap, got %s", res.Project.ID)
	}

	// 5. Detect around test
	detected, err := d.DetectAround(libftDir)
	if err != nil {
		t.Fatalf("DetectAround failed: %v", err)
	}
	if len(detected) == 0 {
		t.Errorf("Expected DetectAround to find nearby sibling projects in tmpDir")
	}
}

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
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Local directory not found: %s, skipping", absPath)
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
