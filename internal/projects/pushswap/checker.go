package pushswap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// StackSimulator simulates push_swap stack operations.
type StackSimulator struct {
	A []int
	B []int
}

// NewSimulator initializes stack A with initial elements (top at index 0) and stack B empty.
func NewSimulator(initial []int) *StackSimulator {
	a := make([]int, len(initial))
	copy(a, initial)
	return &StackSimulator{
		A: a,
		B: make([]int, 0),
	}
}

// Execute applies a single push_swap instruction to stacks A and B.
func (s *StackSimulator) Execute(op string) error {
	switch op {
	case "sa":
		if len(s.A) >= 2 {
			s.A[0], s.A[1] = s.A[1], s.A[0]
		}
	case "sb":
		if len(s.B) >= 2 {
			s.B[0], s.B[1] = s.B[1], s.B[0]
		}
	case "ss":
		if len(s.A) >= 2 {
			s.A[0], s.A[1] = s.A[1], s.A[0]
		}
		if len(s.B) >= 2 {
			s.B[0], s.B[1] = s.B[1], s.B[0]
		}
	case "pa":
		if len(s.B) > 0 {
			top := s.B[0]
			s.B = s.B[1:]
			s.A = append([]int{top}, s.A...)
		}
	case "pb":
		if len(s.A) > 0 {
			top := s.A[0]
			s.A = s.A[1:]
			s.B = append([]int{top}, s.B...)
		}
	case "ra":
		if len(s.A) >= 2 {
			top := s.A[0]
			s.A = append(s.A[1:], top)
		}
	case "rb":
		if len(s.B) >= 2 {
			top := s.B[0]
			s.B = append(s.B[1:], top)
		}
	case "rr":
		if len(s.A) >= 2 {
			top := s.A[0]
			s.A = append(s.A[1:], top)
		}
		if len(s.B) >= 2 {
			top := s.B[0]
			s.B = append(s.B[1:], top)
		}
	case "rra":
		if len(s.A) >= 2 {
			last := s.A[len(s.A)-1]
			s.A = append([]int{last}, s.A[:len(s.A)-1]...)
		}
	case "rrb":
		if len(s.B) >= 2 {
			last := s.B[len(s.B)-1]
			s.B = append([]int{last}, s.B[:len(s.B)-1]...)
		}
	case "rrr":
		if len(s.A) >= 2 {
			last := s.A[len(s.A)-1]
			s.A = append([]int{last}, s.A[:len(s.A)-1]...)
		}
		if len(s.B) >= 2 {
			last := s.B[len(s.B)-1]
			s.B = append([]int{last}, s.B[:len(s.B)-1]...)
		}
	default:
		return fmt.Errorf("invalid instruction: %q", op)
	}
	return nil
}

// Verify checks that Stack B is empty and Stack A is sorted and preserves elements.
func (s *StackSimulator) Verify(initial []int) error {
	if len(s.B) > 0 {
		return fmt.Errorf("stack B is not empty after execution (%d elements remaining: %v)", len(s.B), s.B)
	}
	if len(s.A) != len(initial) {
		return fmt.Errorf("stack A size mismatch: expected %d, got %d", len(initial), len(s.A))
	}
	for i := 1; i < len(s.A); i++ {
		if s.A[i-1] > s.A[i] {
			return fmt.Errorf("stack A is not sorted: index %d (%d) > index %d (%d)", i-1, s.A[i-1], i, s.A[i])
		}
	}

	origCopy := make([]int, len(initial))
	copy(origCopy, initial)
	sort.Ints(origCopy)

	aCopy := make([]int, len(s.A))
	copy(aCopy, s.A)
	sort.Ints(aCopy)

	for i := range origCopy {
		if origCopy[i] != aCopy[i] {
			return fmt.Errorf("stack elements modified: expected %v, got %v", origCopy, aCopy)
		}
	}
	return nil
}

// RunInstructions parses push_swap stdout, executes each operation, and validates the result.
func RunInstructions(initial []int, rawOutput string) (int, error) {
	sim := NewSimulator(initial)
	trimmed := strings.TrimSpace(rawOutput)
	if trimmed == "" {
		if err := sim.Verify(initial); err != nil {
			return 0, err
		}
		return 0, nil
	}

	lines := strings.Split(trimmed, "\n")
	opCount := 0
	for idx, line := range lines {
		op := strings.TrimSpace(line)
		if op == "" {
			continue
		}
		if err := sim.Execute(op); err != nil {
			return opCount, fmt.Errorf("line %d: %w", idx+1, err)
		}
		opCount++
	}

	if err := sim.Verify(initial); err != nil {
		return opCount, err
	}
	return opCount, nil
}

// ParseArgs flattens space-separated or individual string arguments into a slice of ints.
func ParseArgs(args []string) ([]int, error) {
	var nums []int
	for _, arg := range args {
		parts := strings.Fields(arg)
		for _, p := range parts {
			val, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid integer %q: %w", p, err)
			}
			nums = append(nums, val)
		}
	}
	return nums, nil
}
