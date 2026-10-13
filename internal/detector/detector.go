package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristan-gscn/repaco/internal/model"
)

// KnownProjects catalogues the supported 42 projects and their signatures.
var KnownProjects = map[model.ProjectID]model.ProjectMetadata{
	model.ProjectLibft: {
		ID:             model.ProjectLibft,
		Name:           "Libft",
		Description:    "Your very first own C standard library",
		SubjectVersion: "19.3",
		HasMakefile:    true,
		LibraryName:    "libft.a",
		RequiredFiles: []string{
			"libft.h", "Makefile",
			"ft_isalpha.c", "ft_isdigit.c", "ft_isalnum.c", "ft_isascii.c", "ft_isprint.c",
			"ft_strlen.c", "ft_memset.c", "ft_bzero.c", "ft_memcpy.c", "ft_memmove.c",
			"ft_strlcpy.c", "ft_strlcat.c", "ft_toupper.c", "ft_tolower.c", "ft_strchr.c",
			"ft_strrchr.c", "ft_strncmp.c", "ft_memchr.c", "ft_memcmp.c", "ft_strnstr.c",
			"ft_atoi.c", "ft_calloc.c", "ft_strdup.c", "ft_substr.c", "ft_strjoin.c",
			"ft_strtrim.c", "ft_split.c", "ft_itoa.c", "ft_strmapi.c", "ft_striteri.c",
			"ft_putchar_fd.c", "ft_putstr_fd.c", "ft_putendl_fd.c", "ft_putnbr_fd.c",
		},
		BonusFiles: []string{
			"ft_lstnew_bonus.c", "ft_lstadd_front_bonus.c", "ft_lstsize_bonus.c",
			"ft_lstlast_bonus.c", "ft_lstadd_back_bonus.c", "ft_lstdelone_bonus.c",
			"ft_lstclear_bonus.c", "ft_lstiter_bonus.c", "ft_lstmap_bonus.c",
		},
	},
	model.ProjectFtPrintf: {
		ID:             model.ProjectFtPrintf,
		Name:           "ft_printf",
		Description:    "Recode printf: variadic arguments and format specifiers",
		SubjectVersion: "12.1",
		HasMakefile:    true,
		LibraryName:    "libftprintf.a",
		RequiredFiles: []string{
			"Makefile",
		},
		BonusFiles: []string{},
	},
	model.ProjectGetNextLine: {
		ID:             model.ProjectGetNextLine,
		Name:           "get_next_line",
		Description:    "Function reading a line terminated by newline from a file descriptor",
		SubjectVersion: "14.3",
		HasMakefile:    false,
		RequiredFiles: []string{
			"get_next_line.h", "get_next_line.c", "get_next_line_utils.c",
		},
		BonusFiles: []string{
			"get_next_line_bonus.h", "get_next_line_bonus.c", "get_next_line_utils_bonus.c",
		},
	},
	model.ProjectPushSwap: {
		ID:             model.ProjectPushSwap,
		Name:           "push_swap",
		Description:    "Sort data on stacks using minimal operations and algorithmic complexity",
		SubjectVersion: "1.1",
		HasMakefile:    true,
		BinaryName:     "push_swap",
		RequiredFiles: []string{
			"Makefile", "push_swap.h",
		},
		BonusFiles: []string{
			"checker.h",
		},
	},
}

// Detector analyzes directory structure to identify which 42 project it contains.
type Detector struct{}

// New creates a new Detector instance.
func New() *Detector {
	return &Detector{}
}

// Detect inspects rootDir and returns the best matching DetectionResult.
func (d *Detector) Detect(rootDir string) (*model.DetectionResult, error) {
	absDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDir, err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("directory not found: %s", absDir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absDir)
	}

	files, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f.Name()] = true
	}

	// 1. Check get_next_line: unique signature is get_next_line.h/c
	if fileMap["get_next_line.h"] && fileMap["get_next_line.c"] {
		meta := KnownProjects[model.ProjectGetNextLine]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  1.0,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 2. Check push_swap: unique signature is push_swap.h or Makefile producing push_swap
	if fileMap["push_swap.h"] || hasSubfile(absDir, "includes", "push_swap.h") {
		meta := KnownProjects[model.ProjectPushSwap]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  1.0,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 3. Check ft_printf: check ft_printf.h / ft_printf.c / Makefile with libftprintf.a
	isFtPrintf := false
	if fileMap["ft_printf.h"] || hasSubfile(absDir, "includes", "ft_printf.h") ||
		fileMap["ft_printf.c"] || hasSubfile(absDir, "src", "ft_printf.c") {
		isFtPrintf = true
	} else if fileMap["Makefile"] {
		if content, err := os.ReadFile(filepath.Join(absDir, "Makefile")); err == nil {
			if strings.Contains(string(content), "libftprintf.a") || strings.Contains(string(content), "ft_printf") {
				isFtPrintf = true
			}
		}
	}
	if isFtPrintf {
		meta := KnownProjects[model.ProjectFtPrintf]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  0.95,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 4. Check libft: libft.h present + Makefile or ft_*.c
	if fileMap["libft.h"] || hasSubfile(absDir, "includes", "libft.h") {
		meta := KnownProjects[model.ProjectLibft]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  0.95,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// If no match found
	return &model.DetectionResult{
		Project: model.ProjectMetadata{
			ID:          model.ProjectUnknown,
			Name:        "Unknown",
			Description: "Could not identify a supported 42 project",
		},
		Confidence: 0.0,
		RootDir:    absDir,
	}, nil
}

// DetectAround scans immediate subdirectories and parent siblings to discover 42 projects nearby.
func (d *Detector) DetectAround(rootDir string) ([]model.DiscoveredProject, error) {
	absDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	var candidates []string
	seen := make(map[string]bool)
	seen[absDir] = true

	// 1. Immediate subdirectories of rootDir
	if entries, err := os.ReadDir(absDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				p := filepath.Join(absDir, e.Name())
				if !seen[p] {
					seen[p] = true
					candidates = append(candidates, p)
				}
			}
		}
	}

	// 2. Sibling directories in parent folder
	parent := filepath.Dir(absDir)
	if entries, err := os.ReadDir(parent); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				p := filepath.Join(parent, e.Name())
				if !seen[p] {
					seen[p] = true
					candidates = append(candidates, p)
				}
			}
		}
	}

	var discovered []model.DiscoveredProject
	for _, c := range candidates {
		res, err := d.Detect(c)
		if err == nil && res.Project.ID != model.ProjectUnknown {
			rel, err := filepath.Rel(absDir, c)
			if err != nil {
				rel = c
			} else if !strings.HasPrefix(rel, ".") {
				rel = "./" + rel
			}
			discovered = append(discovered, model.DiscoveredProject{
				Detection: *res,
				RelPath:   rel,
				AbsPath:   c,
			})
		}
	}

	return discovered, nil
}

func (d *Detector) populateFiles(rootDir string, meta model.ProjectMetadata, fileMap map[string]bool, res *model.DetectionResult) {
	for _, req := range meta.RequiredFiles {
		if fileMap[req] || fileExistsInTree(rootDir, req) {
			res.FoundFiles = append(res.FoundFiles, req)
		} else {
			res.MissingFiles = append(res.MissingFiles, req)
		}
	}

	bonusFound := 0
	for _, b := range meta.BonusFiles {
		if fileMap[b] || fileExistsInTree(rootDir, b) {
			bonusFound++
		}
	}
	res.HasBonus = bonusFound > 0
}

func hasSubfile(rootDir, subDir, fileName string) bool {
	p := filepath.Join(rootDir, subDir, fileName)
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func fileExistsInTree(rootDir, fileName string) bool {
	var found bool
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return nil
		}
		if !info.IsDir() && info.Name() == fileName {
			found = true
		}
		return nil
	})
	return found
}
