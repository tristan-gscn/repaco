package diff

import (
	"strings"
	"testing"
)

func TestEscapeInvisible(t *testing.T) {
	input := "hello\x00world\n\t"
	escaped := EscapeInvisible(input)
	if !strings.Contains(escaped, "\\0") {
		t.Errorf("Expected \\0 in output, got: %s", escaped)
	}
	if !strings.Contains(escaped, "\\t") {
		t.Errorf("Expected \\t in output, got: %s", escaped)
	}
}

func TestCompare(t *testing.T) {
	diff := Compare("line 1\nline 2", "line 1\nline 2 modified")
	if !strings.Contains(diff, "- line 2") || !strings.Contains(diff, "+ line 2 modified") {
		t.Errorf("Expected diff markers, got:\n%s", diff)
	}
}
