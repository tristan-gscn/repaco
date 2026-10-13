# Contributing to repaco

Thank you for your interest in contributing to **repaco**! 

`repaco` was built by and for 42 school students. One of the primary design goals is to keep the codebase **simple, declarative, and accessible**, so that anyone—even if you are relatively new to Go—can easily contribute test cases or add support for new 42 projects.

---

## Architecture in a Nutshell

`repaco` follows a clear, decoupled architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                          cmd/repaco                         │
│                    (CLI entrypoint & flags)                 │
└──────────────────────────────┬──────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                ▼                             ▼
     internal/detector              internal/engine
     (identifies 42 project)        (coordinates test run)
                │                             │
                └──────────────┬──────────────┘
                               ▼
                    internal/projects/<name>/
      ┌────────────────────────┬────────────────────────┐
      │                        │                        │
      ▼                        ▼                        ▼
   cases.go                suite.go               validator.go
(declarative test       (build & runner         (simulators & logic
 case definitions)       coordination)           verification)
```

1. **`internal/detector`**: Examines a target directory (looking for headers like `libft.h`, target binaries like `push_swap` or `codexion`, etc.) to automatically detect which 42 project it is.
2. **`internal/projects/<name>`**: The test suite for each specific project:
   - **`cases.go`**: Pure, declarative lists of test cases (inputs, categories, timeouts, expected behavior). **This is where most contributors add tests!**
   - **`suite.go`**: Implements the `projects.Suite` interface (`Prepare`, `Plan`, `RunCase`, `Cleanup`).
   - Optional simulator/checker (e.g. `checker.go` in push_swap, `validator.go` in codexion).
3. **`internal/builder` & `internal/runner`**: Handles invoking `make`, running binaries with timeouts, managing processes, capturing stdout/stderr, and running memory checks (`leaks` / `valgrind`).
4. **`internal/ui`**: Bubbletea-based interactive TUI, or headless markdown/JSON output.

---

## 1. Adding a Test Case to an Existing Project

Adding a new test case to projects like `push_swap` or `codexion` takes less than 2 minutes and requires zero Go knowledge beyond editing a struct.

### Example: Adding an edge case to `push_swap`

1. Open [`internal/projects/pushswap/cases.go`](internal/projects/pushswap/cases.go).
2. Locate the category you want to add to (e.g., `errorDefs` or `smallDefs`).
3. Add your entry:

```go
// In errorDefs inside GetDefaultCases():
{"err_my_new_edge_case", "Trailing space after sign", "Number with trailing space '+ 42' must write 'Error\\n'", []string{"+ 42"}},
```

4. Run the suite tests to make sure everything parses:
```bash
go test -v ./internal/projects/pushswap/...
```

### Example: Adding a simulation test to `codexion`

1. Open [`internal/projects/codexion/cases.go`](internal/projects/codexion/cases.go).
2. Find the relevant slice (`syntaxCases`, `logicCases`, `nominalCases`, `burnoutCases`, or `stressCases`).
3. Add your test definition:

```go
// In syntaxCases:
{"syntax_hex_arg", "Hexadecimal argument ('0x42')", "Hexadecimal parameter must be safely rejected", []string{"4", "0x42", "200", "200", "200", "5", "10", "fifo"}},
```

4. Run the suite tests:
```bash
go test -v ./internal/projects/codexion/...
```

That's it! When you run `repaco`, your test case will immediately show up in the TUI checklist.

---

## 2. Writing Good Test Cases

When writing test cases, please follow these principles:

1. **Helpful Explanations & Advice**:
   - Don't just say *"Test failed"*. Explain *why* it failed and what the student can check:
   - *Good*: `Advice: "Ensure ft_atoi detects numbers exceeding INT_MAX (2147483647)."`
   - *Bad*: `Advice: "Fix your code."`
2. **Deterministic & Isolated**:
   - Tests should never leave temporary files behind or depend on previous tests having executed first.
   - Use reasonable timeouts (typically 2 to 5 seconds).
3. **Categorization**:
   - Pick the appropriate `model.Category`:
     - `CategoryMandatory`: Basic requirements of the 42 subject.
     - `CategoryEdgeCase`: Tricky inputs, overflow, empty strings, null pointers.
     - `CategoryBenchmark`: Performance or operation count targets (e.g. push_swap `< 700` ops).
     - `CategoryMemory`: Leak checks and address sanitizer tests.
     - `CategoryBonus`: Features belonging to the project's bonus section.

---

## 3. Adding Support for a New 42 Project

Want to add a brand new 42 project (e.g. `minitalk`, `pipex`, `so_long`, `philosophers`)? Here is the step-by-step checklist:

### Step 1: Register the Project ID

Open [`internal/model/project.go`](internal/model/project.go) and add your project constant:

```go
const (
    // ...
    ProjectPipex ProjectID = "pipex"
)
```

### Step 2: Add Detection Heuristics

Open [`internal/detector/detector.go`](internal/detector/detector.go) and add project metadata and detection rules in `KnownProjects`:

```go
KnownProjects[model.ProjectPipex] = model.ProjectMetadata{
    ID:          model.ProjectPipex,
    Name:        "pipex",
    Description: "UNIX mechanisms: pipes, redirections, and child processes",
    DetectFiles: []string{"pipex.h", "pipex.c"},
}
```

### Step 3: Create the Project Package

Create a new directory `internal/projects/<your_project>/`:

1. **`cases.go`**: Declarative list of test cases:
```go
package myproject

import (
    "time"
    "github.com/tristan-gscn/repaco/internal/model"
)

type TestCaseDef struct {
    ID          string
    Name        string
    Category    model.Category
    Description string
    Args        []string
    Timeout     time.Duration
}

func GetDefaultCases() []TestCaseDef {
    return []TestCaseDef{
        {
            ID:          "myproject_basic",
            Name:        "Basic execution",
            Category:    model.CategoryMandatory,
            Description: "Runs basic test case",
            Args:        []string{"arg1", "arg2"},
            Timeout:     3 * time.Second,
        },
    }
}
```

2. **`suite.go`**: Implement the `projects.Suite` interface:
```go
package myproject

import (
    "context"
    "fmt"
    "path/filepath"
    "time"

    "github.com/tristan-gscn/repaco/internal/builder"
    "github.com/tristan-gscn/repaco/internal/config"
    "github.com/tristan-gscn/repaco/internal/detector"
    "github.com/tristan-gscn/repaco/internal/model"
    "github.com/tristan-gscn/repaco/internal/projects"
    "github.com/tristan-gscn/repaco/internal/runner"
)

type Suite struct {
    binPath string
    cases   map[string]TestCaseDef
}

func init() {
    projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
    return model.ProjectMyProject
}

func (s *Suite) Metadata() model.ProjectMetadata {
    return detector.KnownProjects[model.ProjectMyProject]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
    // 1. Check Makefile & compile binary with -Wall -Wextra -Werror
    out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
    if err != nil {
        return &model.TestResult{
            Case: model.TestCase{
                ID:       "myproject_build",
                Name:     "make all compilation",
                Category: model.CategoryBuild,
            },
            Status:      model.StatusFailed,
            Explanation: fmt.Sprintf("Compilation failed:\n%s", out),
            Advice:      "Fix compiler errors and ensure Makefile compiles cleanly.",
        }, nil
    }
    s.binPath = filepath.Join(dir, "myproject")
    return &model.TestResult{
        Case: model.TestCase{
            ID:       "myproject_build",
            Name:     "Build verification",
            Category: model.CategoryBuild,
        },
        Status: model.StatusPassed,
    }, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
    defs := GetDefaultCases()
    s.cases = make(map[string]TestCaseDef)
    var plan []model.TestCase
    for _, d := range defs {
        s.cases[d.ID] = d
        plan = append(plan, model.TestCase{
            ID:          d.ID,
            Name:        d.Name,
            Description: d.Description,
            Category:    d.Category,
            Timeout:     d.Timeout,
        })
    }
    return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
    def := s.cases[tc.ID]
    res, err := r.Run(ctx, s.binPath, runner.RunOptions{
        Args:    def.Args,
        Cwd:     dir,
        Timeout: def.Timeout,
    })
    if err != nil {
        return &model.TestResult{
            Case:        tc,
            Status:      model.StatusFailed,
            Explanation: err.Error(),
        }, nil
    }

    if res.ExitCode != 0 {
        return &model.TestResult{
            Case:        tc,
            Status:      model.StatusFailed,
            Explanation: fmt.Sprintf("Unexpected exit code: %d", res.ExitCode),
        }, nil
    }

    return &model.TestResult{
        Case:   tc,
        Status: model.StatusPassed,
    }, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
    return nil
}
```

3. Register the import in [`cmd/repaco/main.go`](cmd/repaco/main.go) so the package initializes:
```go
import (
    // ...
    _ "github.com/tristan-gscn/repaco/internal/projects/myproject"
)
```

---

## 4. Local Development Workflow

### Prerequisites

- Go 1.21 or higher
- C compiler (`clang` or `gcc`)
- Make

### Running Tests

Run all unit tests across the repository:
```bash
go test -v ./...
```

Run tests for a single package:
```bash
go test -v ./internal/projects/pushswap/...
```

### Building the Binary

Compile `repaco` locally:
```bash
# Using Makefile
make build

# Or directly with Go
go build -o repaco ./cmd/repaco
```

### Testing on Real 42 Repositories

Test your changes against actual 42 projects:

```bash
# Interactive TUI mode
./repaco /path/to/student/project

# Headless agent report (convenient for debugging exact outputs)
./repaco --agent /path/to/student/project
```

---

## 5. Submitting a Pull Request

1. Fork the repository and create your branch from `main`:
   ```bash
   git checkout -b feature/my-new-tests
   ```
2. Commit your changes with clear commit messages:
   ```bash
   git commit -m "feat(pushswap): add INT_MIN edge case test"
   ```
3. Ensure all tests pass:
   ```bash
   go test ./...
   ```
4. Push your branch and open a Pull Request on GitHub.

Thank you for helping make `repaco` better for all 42 students!
