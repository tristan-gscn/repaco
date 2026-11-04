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
			"ft_lstnew.c", "ft_lstadd_front.c", "ft_lstsize.c",
			"ft_lstlast.c", "ft_lstadd_back.c", "ft_lstdelone.c",
			"ft_lstclear.c", "ft_lstiter.c", "ft_lstmap.c",
		},
		BonusFiles: []string{},
	},
	model.ProjectFtPrintf: {
		ID:             model.ProjectFtPrintf,
		Name:           "ft_printf",
		Description:    "Recode printf: variadic arguments and format specifiers",
		SubjectVersion: "12.1",
		HasMakefile:    true,
		LibraryName:    "libftprintf.a",
		RequiredFiles: []string{
			"Makefile", "ft_printf.h",
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
			"get_next_line_bonus.h", "get_next_line_bonus.c", "get_next_line_utils_bonus.c",
		},
		BonusFiles: []string{},
	},
	model.ProjectPushSwap: {
		ID:             model.ProjectPushSwap,
		Name:           "push_swap",
		Description:    "Sort data on stacks using minimal operations and algorithmic complexity",
		SubjectVersion: "1.1",
		HasMakefile:    true,
		BinaryName:     "push_swap",
		RequiredFiles: []string{
			"Makefile", "push_swap.h", "checker.h",
		},
		BonusFiles: []string{},
	},
	model.ProjectCodexion: {
		ID:             model.ProjectCodexion,
		Name:           "codexion",
		Description:    "High-precision multi-threaded simulation and concurrency scheduler",
		SubjectVersion: "1.0",
		HasMakefile:    true,
		BinaryName:     "codexion",
		RequiredFiles: []string{
			"Makefile", "codexion.h",
		},
		BonusFiles: []string{},
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

	// 1. Check get_next_line: signature is any get_next_line header or source file
	if fileExistsInTree(absDir, "get_next_line.h") || fileExistsInTree(absDir, "get_next_line.c") ||
		fileExistsInTree(absDir, "get_next_line_utils.c") || fileExistsInTree(absDir, "get_next_line_bonus.h") ||
		fileExistsInTree(absDir, "get_next_line_bonus.c") || fileExistsInTree(absDir, "get_next_line_utils_bonus.c") {
		meta := KnownProjects[model.ProjectGetNextLine]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  1.0,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"] || fileMap["makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 2. Check codexion: unique signature is codexion.h or Makefile producing codexion
	isCodexion := false
	if fileExistsInTree(absDir, "codexion.h") {
		isCodexion = true
	} else if fileMap["Makefile"] || fileMap["makefile"] {
		for _, mf := range []string{"Makefile", "makefile"} {
			if content, err := os.ReadFile(filepath.Join(absDir, mf)); err == nil {
				if strings.Contains(string(content), "codexion") {
					isCodexion = true
					break
				}
			}
		}
	}
	if isCodexion {
		meta := KnownProjects[model.ProjectCodexion]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  1.0,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"] || fileMap["makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 3. Check push_swap: unique signature is push_swap.h, checker.h, or Makefile producing push_swap
	isPushSwap := false
	if fileExistsInTree(absDir, "push_swap.h") || fileExistsInTree(absDir, "checker.h") {
		isPushSwap = true
	} else if fileMap["Makefile"] || fileMap["makefile"] {
		for _, mf := range []string{"Makefile", "makefile"} {
			if content, err := os.ReadFile(filepath.Join(absDir, mf)); err == nil {
				if strings.Contains(string(content), "push_swap") || strings.Contains(string(content), "checker") {
					isPushSwap = true
					break
				}
			}
		}
	}
	if isPushSwap {
		meta := KnownProjects[model.ProjectPushSwap]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  1.0,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"] || fileMap["makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 4. Check ft_printf: check ft_printf.h / ft_printf.c / Makefile with libftprintf.a
	isFtPrintf := false
	if fileExistsInTree(absDir, "ft_printf.h") || fileExistsInTree(absDir, "ft_printf.c") {
		isFtPrintf = true
	} else if fileMap["Makefile"] || fileMap["makefile"] {
		for _, mf := range []string{"Makefile", "makefile"} {
			if content, err := os.ReadFile(filepath.Join(absDir, mf)); err == nil {
				if strings.Contains(string(content), "libftprintf.a") || strings.Contains(string(content), "ft_printf") {
					isFtPrintf = true
					break
				}
			}
		}
	}
	if isFtPrintf {
		meta := KnownProjects[model.ProjectFtPrintf]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  0.95,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"] || fileMap["makefile"],
		}
		d.populateFiles(absDir, meta, fileMap, res)
		return res, nil
	}

	// 5. Check libft: libft.h present anywhere, Makefile with libft.a/libft, or multiple ft_*.c functions
	isLibft := false
	if fileExistsInTree(absDir, "libft.h") {
		isLibft = true
	} else if fileMap["Makefile"] || fileMap["makefile"] {
		for _, mf := range []string{"Makefile", "makefile"} {
			if content, err := os.ReadFile(filepath.Join(absDir, mf)); err == nil {
				str := string(content)
				if strings.Contains(str, "libft.a") || strings.Contains(str, "libft") {
					isLibft = true
					break
				}
			}
		}
	}
	if !isLibft {
		ftCount := 0
		for _, f := range KnownProjects[model.ProjectLibft].RequiredFiles {
			if strings.HasPrefix(f, "ft_") && fileExistsInTree(absDir, f) {
				ftCount++
				if ftCount >= 2 {
					isLibft = true
					break
				}
			}
		}
	}
	if isLibft {
		meta := KnownProjects[model.ProjectLibft]
		res := &model.DetectionResult{
			Project:     meta,
			Confidence:  0.95,
			RootDir:     absDir,
			HasMakefile: fileMap["Makefile"] || fileMap["makefile"],
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

func isIgnoredDir(name string) bool {
	switch strings.ToLower(name) {
	case "library", "applications", "system", "node_modules", ".git", ".github",
		"desktop", "documents", "downloads", "movies", "music", "pictures",
		"deriveddata", "build", ".build", ".objects", "public", ".cache", ".repaco":
		return true
	default:
		return false
	}
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

func fileExistsInTree(rootDir, fileName string) bool {
	// 1. Direct check in root and standard source/header subfolders
	subdirs := []string{"", "includes", "inc", "include", "headers", "src", "srcs", "sources"}
	for _, sub := range subdirs {
		var p string
		if sub == "" {
			p = filepath.Join(rootDir, fileName)
		} else {
			p = filepath.Join(rootDir, sub, fileName)
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return true
		}
	}

	// 2. Walk only inside standard source/include directories (e.g., src/string/ft_strlen.c)
	for _, sub := range []string{"src", "srcs", "sources", "includes", "inc", "include"} {
		subPath := filepath.Join(rootDir, sub)
		if info, err := os.Stat(subPath); err == nil && info.IsDir() {
			found := false
			_ = filepath.WalkDir(subPath, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if entry.IsDir() {
					name := entry.Name()
					if strings.HasPrefix(name, ".") || isIgnoredDir(name) {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.EqualFold(entry.Name(), fileName) {
					found = true
					return filepath.SkipAll
				}
				return nil
			})
			if found {
				return true
			}
		}
	}

	return false
}
