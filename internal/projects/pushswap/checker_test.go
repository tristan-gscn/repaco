package pushswap

import (
	"testing"
)

func TestSimulatorBasicOps(t *testing.T) {
	initial := []int{3, 2, 1}
	sim := NewSimulator(initial)

	// sa
	if err := sim.Execute("sa"); err != nil {
		t.Fatalf("sa failed: %v", err)
	}
	if sim.A[0] != 2 || sim.A[1] != 3 || sim.A[2] != 1 {
		t.Fatalf("unexpected A after sa: %v", sim.A)
	}

	// rra -> [1, 2, 3]
	if err := sim.Execute("rra"); err != nil {
		t.Fatalf("rra failed: %v", err)
	}
	if err := sim.Verify(initial); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestSimulatorRunInstructions(t *testing.T) {
	initial := []int{2, 1, 3}
	ops := "sa\n"
	count, err := RunInstructions(initial, ops)
	if err != nil {
		t.Fatalf("run instructions failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	// Unsorted with empty instructions should fail
	_, err = RunInstructions([]int{2, 1}, "")
	if err == nil {
		t.Fatalf("expected error for unsorted with no ops, got nil")
	}

	// Invalid instruction
	_, err = RunInstructions([]int{1, 2}, "invalid_op\n")
	if err == nil {
		t.Fatalf("expected error for invalid op, got nil")
	}
}
