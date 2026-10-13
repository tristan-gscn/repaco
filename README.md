# repaco

> **Universal 42 Project Tester** — Lightning-fast, zero-config test runner for 42 School projects featuring an interactive JetBrains-style TUI and headless AI agent diagnostics.

---

## ⚡ Installation

Install the latest pre-compiled binary for your system (macOS / Linux, x86_64 / ARM64):

```bash
curl -fsSL https://raw.githubusercontent.com/tristan-gscn/repaco/main/install.sh | bash
```

*The binary is installed to `~/.local/bin/repaco`.*

---

## 🚀 Usage

Navigate to any 42 project repository and run:

```bash
# Interactive TUI mode (auto-detects project in current directory)
repaco

# Run tests on a specific project directory
repaco ~/42/push-swap

# Headless AI Agent mode (outputs Markdown report with exact C snippets and diffs)
repaco --agent

# Machine-readable JSON output for CI/CD pipelines
repaco --json
```

---

## 📦 Supported Projects

| Project | Coverage |
| :--- | :--- |
| **libft** | Full mandatory & bonus suite, Makefile compliance, edge cases |
| **ft_printf** | Output capture, zero-byte handling, conversion specifiers, AddressSanitizer |
| **get_next_line** | Multi-`BUFFER_SIZE` compilation (1, 42, 10M), EOF repeat, pipe stdin, bonus multi-fd |
| **push_swap** | Pure-Go simulator, error management, small sort $\le 12$ ops, 100/500 benchmarks with 42 grading |

---

## 🎯 Highlights

- **Auto-Detection**: Automatically detects your 42 project and applies the right test battery.
- **Modern JetBrains TUI**: Tree navigation, collapsed-by-default passed tests, and dual-tone pass/fail progress bar.
- **Actionable Diagnostics**: Detailed memory leak reports, crash signal traces, diff views, and standalone reproduction commands.
