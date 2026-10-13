package codexion

import (
	"testing"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestBurnoutLatencyOK(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  100,
		TimeToCompile:  50,
		TimeToDebug:    50,
		TimeToRefactor: 50,
		NumCompiles:    1,
		DongleCooldown: 10,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 105, Coder: 2, State: "burned out"}, // 5ms lateness <= 10ms target
	}
	res := ValidateSimulation(params, logs, true, false, nil)
	if res.Status != model.StatusPassed {
		t.Fatalf("Expected PASS, got %s: %s", res.Status, res.Explanation)
	}
	if len(res.Warnings) > 0 {
		t.Errorf("Expected 0 warnings, got %d", len(res.Warnings))
	}
}

func TestBurnoutLatencyWarn(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  100,
		TimeToCompile:  50,
		TimeToDebug:    50,
		TimeToRefactor: 50,
		NumCompiles:    1,
		DongleCooldown: 10,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 115, Coder: 2, State: "burned out"}, // 15ms lateness: >10ms and <=25ms
	}
	res := ValidateSimulation(params, logs, true, false, nil)
	if res.Status != model.StatusPassed {
		t.Fatalf("Expected PASS with warning, got %s: %s", res.Status, res.Explanation)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(res.Warnings))
	}
}

func TestBurnoutLatencyKO(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  100,
		TimeToCompile:  50,
		TimeToDebug:    50,
		TimeToRefactor: 50,
		NumCompiles:    1,
		DongleCooldown: 10,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 130, Coder: 2, State: "burned out"}, // 30ms lateness > 25ms
	}
	res := ValidateSimulation(params, logs, true, false, nil)
	if res.Status != model.StatusFailed {
		t.Fatalf("Expected FAIL, got %s", res.Status)
	}
}

func TestAdjacentCodersCannotCompileSimultaneously(t *testing.T) {
	params := &CaseParams{
		NumCoders:      4,
		TimeToBurnout:  500,
		TimeToCompile:  100,
		TimeToDebug:    100,
		TimeToRefactor: 100,
		NumCompiles:    1,
		DongleCooldown: 10,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 1, Coder: 2, State: "has taken a dongle"},
		{LineNo: 5, TS: 1, Coder: 2, State: "has taken a dongle"},
		{LineNo: 6, TS: 1, Coder: 2, State: "is compiling"}, // Coder 1 and 2 adjacent compiling
	}
	res := ValidateSimulation(params, logs, false, true, nil)
	if res.Status != model.StatusFailed {
		t.Fatalf("Expected FAIL for adjacent compiling, got %s", res.Status)
	}
}

func TestDongleCooldownEnforcement(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  1000,
		TimeToCompile:  50,
		TimeToDebug:    10,
		TimeToRefactor: 10,
		NumCompiles:    2,
		DongleCooldown: 50,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 50, Coder: 1, State: "is debugging"}, // dongles cool down until 50+50 = 100ms
		{LineNo: 5, TS: 55, Coder: 2, State: "has taken a dongle"},
		{LineNo: 6, TS: 60, Coder: 2, State: "has taken a dongle"},
		{LineNo: 7, TS: 70, Coder: 2, State: "is compiling"}, // 70ms < 100ms cooldown -> KO
	}
	res := ValidateSimulation(params, logs, false, true, nil)
	if res.Status != model.StatusFailed {
		t.Fatalf("Expected FAIL for cooldown violation, got %s", res.Status)
	}
}

func TestDongleCooldownJitterWarn(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  1000,
		TimeToCompile:  50,
		TimeToDebug:    10,
		TimeToRefactor: 10,
		NumCompiles:    1,
		DongleCooldown: 50,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 50, Coder: 1, State: "is debugging"}, // cooldown until 100ms
		{LineNo: 5, TS: 90, Coder: 2, State: "has taken a dongle"},
		{LineNo: 6, TS: 95, Coder: 2, State: "has taken a dongle"},
		{LineNo: 7, TS: 97, Coder: 2, State: "is compiling"}, // 97ms: only 3ms before 100ms (<= 10ms tolerance)
		{LineNo: 8, TS: 147, Coder: 2, State: "is debugging"},
		{LineNo: 9, TS: 157, Coder: 2, State: "is refactoring"},
		{LineNo: 10, TS: 157, Coder: 1, State: "is refactoring"},
	}
	res := ValidateSimulation(params, logs, false, true, nil)
	if res.Status != model.StatusPassed {
		t.Fatalf("Expected PASS with warning, got %s: %s", res.Status, res.Explanation)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("Expected at least 1 warning for jitter")
	}
}

func TestPostMortemActionsInvalid(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  100,
		TimeToCompile:  50,
		TimeToDebug:    50,
		TimeToRefactor: 50,
		NumCompiles:    1,
		DongleCooldown: 10,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 3, TS: 0, Coder: 1, State: "is compiling"},
		{LineNo: 4, TS: 105, Coder: 2, State: "burned out"},
		{LineNo: 5, TS: 110, Coder: 1, State: "is debugging"}, // Post-mortem action logged after burnout
	}
	res := ValidateSimulation(params, logs, true, false, nil)
	if res.Status != model.StatusFailed {
		t.Fatalf("Expected FAIL for post-mortem action, got %s", res.Status)
	}
}

func TestCompileWithoutTwoTakesIsInvalid(t *testing.T) {
	params := &CaseParams{
		NumCoders:      2,
		TimeToBurnout:  1000,
		TimeToCompile:  10,
		TimeToDebug:    5,
		TimeToRefactor: 5,
		NumCompiles:    1,
		DongleCooldown: 0,
		Scheduler:      "fifo",
	}
	logs := []LogEntry{
		{LineNo: 1, TS: 0, Coder: 1, State: "has taken a dongle"},
		{LineNo: 2, TS: 1, Coder: 1, State: "is compiling"}, // Only 1 dongle taken
	}
	res := ValidateSimulation(params, logs, false, false, nil)
	if res.Status != model.StatusFailed {
		t.Fatalf("Expected FAIL for compiling with 1 dongle, got %s", res.Status)
	}
}

func TestErrorCaseValidation(t *testing.T) {
	// Non-zero exit code
	r1 := ValidateErrorCase(1, "", "Error: invalid argument\n")
	if r1.Status != model.StatusPassed {
		t.Errorf("Expected PASS for non-zero exit code, got %s", r1.Status)
	}

	// Exit code 0 with error message
	r2 := ValidateErrorCase(0, "Error: bad scheduler\n", "")
	if r2.Status != model.StatusPassed || len(r2.Warnings) == 0 {
		t.Errorf("Expected PASS with warning, got %s", r2.Status)
	}

	// Exit code 0 without error message
	r3 := ValidateErrorCase(0, "4 coders running...\n", "")
	if r3.Status != model.StatusFailed {
		t.Errorf("Expected FAIL for successful looking output, got %s", r3.Status)
	}
}

func TestParseCaseArgs(t *testing.T) {
	valid := []string{"4", "800", "200", "200", "200", "5", "10", "fifo"}
	p, err := ParseCaseArgs(valid)
	if err != nil {
		t.Fatalf("ParseCaseArgs failed: %v", err)
	}
	if p.NumCoders != 4 || p.TimeToBurnout != 800 || p.Scheduler != "fifo" {
		t.Errorf("Parsed mismatch: %+v", p)
	}

	invalid := []string{"-4", "800", "200", "200", "200", "5", "10", "fifo"}
	if _, err := ParseCaseArgs(invalid); err == nil {
		t.Errorf("Expected error for negative coders count")
	}

	invalidSched := []string{"4", "800", "200", "200", "200", "5", "10", "round_robin"}
	if _, err := ParseCaseArgs(invalidSched); err == nil {
		t.Errorf("Expected error for unsupported scheduler")
	}
}
