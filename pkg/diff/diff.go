package diff

import (
	"fmt"
	"strings"
	"unicode"
)

// EscapeInvisible replaces non-printable and whitespace characters with visible representations.
func EscapeInvisible(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString("\\n\n")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		case 0:
			b.WriteString("\\0")
		default:
			if unicode.IsPrint(r) {
				b.WriteRune(r)
			} else {
				b.WriteString(fmt.Sprintf("\\x%02x", r))
			}
		}
	}
	return b.String()
}

// Compare returns a side-by-side or formatted diff comparing expected and actual outputs.
func Compare(expected, actual string) string {
	if expected == actual {
		return "Outputs match exactly."
	}

	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	var sb strings.Builder
	sb.WriteString("--- Expected\n")
	sb.WriteString("+++ Actual\n")

	max := len(expLines)
	if len(actLines) > max {
		max = len(actLines)
	}

	for i := 0; i < max; i++ {
		var expL, actL string
		hasExp := i < len(expLines)
		hasAct := i < len(actLines)

		if hasExp {
			expL = expLines[i]
		}
		if hasAct {
			actL = actLines[i]
		}

		if hasExp && hasAct && expL == actL {
			sb.WriteString(fmt.Sprintf("  %s\n", expL))
		} else {
			if hasExp {
				sb.WriteString(fmt.Sprintf("- %s\n", EscapeInvisible(expL)))
			}
			if hasAct {
				sb.WriteString(fmt.Sprintf("+ %s\n", EscapeInvisible(actL)))
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
