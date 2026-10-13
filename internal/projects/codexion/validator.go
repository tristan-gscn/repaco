package codexion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tristan-gscn/repaco/internal/model"
)

// LogEntry represents a single parsed output line from codexion.
type LogEntry struct {
	LineNo int
	TS     int64
	Coder  int
	State  string
	Raw    string
}

// CaseParams holds parsed CLI arguments for a codexion simulation run.
type CaseParams struct {
	NumCoders      int
	TimeToBurnout  int64
	TimeToCompile  int64
	TimeToDebug    int64
	TimeToRefactor int64
	NumCompiles    int
	DongleCooldown int64
	Scheduler      string
}

// ValidationResult contains the analysis verdict of codexion execution.
type ValidationResult struct {
	Status      model.Status
	Explanation string
	Advice      string
	Expected    string
	Actual      string
	Diff        string
	Warnings    []string
}

// ParseCaseArgs parses the 8 positional arguments of codexion.
func ParseCaseArgs(args []string) (*CaseParams, error) {
	if len(args) != 8 {
		return nil, fmt.Errorf("expected 8 arguments, got %d", len(args))
	}

	nCoders, err := strconv.Atoi(args[0])
	if err != nil || nCoders <= 0 {
		return nil, fmt.Errorf("number_of_coders must be an integer > 0, got '%s'", args[0])
	}

	tBurn, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || tBurn < 0 {
		return nil, fmt.Errorf("time_to_burnout must be >= 0, got '%s'", args[1])
	}

	tComp, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil || tComp < 0 {
		return nil, fmt.Errorf("time_to_compile must be >= 0, got '%s'", args[2])
	}

	tDbg, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil || tDbg < 0 {
		return nil, fmt.Errorf("time_to_debug must be >= 0, got '%s'", args[3])
	}

	tRef, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil || tRef < 0 {
		return nil, fmt.Errorf("time_to_refactor must be >= 0, got '%s'", args[4])
	}

	nComp, err := strconv.Atoi(args[5])
	if err != nil || nComp < 0 {
		return nil, fmt.Errorf("number_of_compiles_required must be >= 0, got '%s'", args[5])
	}

	cooldown, err := strconv.ParseInt(args[6], 10, 64)
	if err != nil || cooldown < 0 {
		return nil, fmt.Errorf("dongle_cooldown must be >= 0, got '%s'", args[6])
	}

	sched := args[7]
	if sched != "fifo" && sched != "edf" {
		return nil, fmt.Errorf("scheduler must be 'fifo' or 'edf', got '%s'", sched)
	}

	return &CaseParams{
		NumCoders:      nCoders,
		TimeToBurnout:  tBurn,
		TimeToCompile:  tComp,
		TimeToDebug:    tDbg,
		TimeToRefactor: tRef,
		NumCompiles:    nComp,
		DongleCooldown: cooldown,
		Scheduler:      sched,
	}, nil
}

var logRegex = regexp.MustCompile(`^(\d+)\s+(\d+)\s+(has taken a dongle|is compiling|is debugging|is refactoring|burned out)$`)

// ParseOutputLogs parses raw stdout lines into structured LogEntry items.
func ParseOutputLogs(raw string, nCoders int) ([]LogEntry, []string, *ValidationResult) {
	lines := strings.Split(raw, "\n")
	var entries []LogEntry
	var warnings []string
	var lastTS int64 = -1

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		m := logRegex.FindStringSubmatch(trimmed)
		if len(m) != 4 {
			return nil, nil, &ValidationResult{
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Invalid log format on line %d: '%s'", idx+1, trimmed),
				Advice:      "Logs must follow '<timestamp_in_ms> <coder_id> <action>' with valid action verbs.",
				Expected:    "<timestamp_ms> <coder_id> (has taken a dongle|is compiling|is debugging|is refactoring|burned out)",
				Actual:      trimmed,
			}
		}

		ts, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil, nil, &ValidationResult{
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Malformed timestamp on line %d: '%s'", idx+1, m[1]),
				Advice:      "Timestamps must be non-negative 64-bit integers in milliseconds.",
			}
		}

		coder, err := strconv.Atoi(m[2])
		if err != nil || coder < 1 || coder > nCoders {
			return nil, nil, &ValidationResult{
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Coder ID out of valid range (1..%d) on line %d: got coder %d", nCoders, idx+1, coder),
				Advice:      "Coder indices are 1-based and must not exceed the total number of coders.",
			}
		}

		if ts < lastTS {
			warnings = append(warnings, fmt.Sprintf("Non-monotonic timestamp on line %d: %dms < previous %dms (possible mutex contention or logging race)", idx+1, ts, lastTS))
		}
		lastTS = ts

		entries = append(entries, LogEntry{
			LineNo: idx + 1,
			TS:     ts,
			Coder:  coder,
			State:  m[3],
			Raw:    trimmed,
		})
	}

	return entries, warnings, nil
}

// ValidateSimulation performs complete concurrency, state machine, timing, and burnout verification.
func ValidateSimulation(params *CaseParams, logs []LogEntry, expectBurnout, expectNoBurnout bool, initialWarnings []string) *ValidationResult {
	warnings := append([]string{}, initialWarnings...)
	n := params.NumCoders

	// 1. Concurrency tracking (Dongles)
	compilingCoders := make(map[int]bool)
	cooldownUntil := make(map[int]int64)  // dongle 1..n
	lastReleasedBy := make(map[int]int)

	// 2. State sequence tracking
	lastState := make(map[int]string)    // "init", "has taken a dongle", "is compiling", etc.
	takenCounts := make(map[int]int)
	for i := 1; i <= n; i++ {
		lastState[i] = "init"
	}

	// 3. Action durations
	lastCompile := make(map[int]int64)
	lastDebug := make(map[int]*int64)
	lastRefactor := make(map[int]*int64)
	compileCounts := make(map[int]int)

	// 4. Burnout tracking
	burnouts := 0

	for idx, log := range logs {
		coder := log.Coder
		ts := log.TS
		state := log.State

		// --- State sequence validation ---
		prev := lastState[coder]
		invalidTransition := false

		if state != "burned out" {
			switch prev {
			case "init":
				invalidTransition = (state != "has taken a dongle")
			case "has taken a dongle":
				if state == "has taken a dongle" {
					invalidTransition = (takenCounts[coder] >= 2)
				} else {
					invalidTransition = (state != "is compiling" || takenCounts[coder] != 2)
				}
			case "is compiling":
				invalidTransition = (state != "is debugging")
			case "is debugging":
				invalidTransition = (state != "is refactoring")
			case "is refactoring":
				invalidTransition = (state != "has taken a dongle")
			}
		}

		if invalidTransition {
			return &ValidationResult{
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Illegal state transition on line %d: coder %d transitioned from '%s' to '%s'", log.LineNo, coder, prev, state),
				Advice:      "Coders must strictly alternate: 2x 'has taken a dongle' -> 'is compiling' -> 'is debugging' -> 'is refactoring'.",
				Expected:    "Valid sequential transition",
				Actual:      fmt.Sprintf("%s -> %s (line %d)", prev, state, log.LineNo),
			}
		}

		// Update state counters
		if state == "has taken a dongle" {
			takenCounts[coder]++

			// Check refactor duration before taking dongles
			if lastRefactor[coder] != nil {
				waited := ts - *lastRefactor[coder]
				if waited < params.TimeToRefactor {
					return &ValidationResult{
						Status:      model.StatusFailed,
						Explanation: fmt.Sprintf("Refactor phase ended too early on line %d: coder %d refactored for only %dms (expected >= %dms)", log.LineNo, coder, waited, params.TimeToRefactor),
						Advice:      "Ensure usleep/simulation loop accurately honors time_to_refactor before attempting to acquire dongles.",
					}
				}
			}
		} else if state == "is compiling" {
			takenCounts[coder] = 0

			// 1 coder cannot compile with 1 dongle
			if n == 1 {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Coder 1 started compiling on line %d, but there is only 1 dongle on the table (2 required)", log.LineNo),
					Advice:      "A simulation with only 1 coder cannot compile because 2 dongles are required; the coder must starve and burn out.",
				}
			}

			// Circular neighbors
			left := coder - 1
			if coder == 1 {
				left = n
			}
			right := coder + 1
			if coder == n {
				right = 1
			}

			// Adjacent coder collision
			if compilingCoders[left] || compilingCoders[right] {
				adj := left
				side := "left"
				if compilingCoders[right] {
					adj = right
					side = "right"
				}
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Concurrency violation on line %d: coder %d started compiling while %s neighbor %d is already compiling", log.LineNo, coder, side, adj),
					Advice:      "Two adjacent coders share a dongle and must never compile simultaneously.",
				}
			}

			// Max simultaneous compilers
			if len(compilingCoders) >= n/2 {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Concurrency violation on line %d: %d coders compiling simultaneously, exceeding maximum capacity (%d)", log.LineNo, len(compilingCoders)+1, n/2),
					Advice:      "With N dongles and 2 dongles per compiler, at most N/2 coders can compile at the same time.",
				}
			}

			// Cooldown check for left dongle and right dongle
			donglesToCheck := []struct {
				id   int
				side string
			}{
				{left, "left"},
				{coder, "right"},
			}
			for _, d := range donglesToCheck {
				if ts < cooldownUntil[d.id] {
					diff := cooldownUntil[d.id] - ts
					if diff > 10 {
						return &ValidationResult{
							Status:      model.StatusFailed,
							Explanation: fmt.Sprintf("Cooldown violation on line %d: coder %d compiled using %s dongle %d at %dms, but cooldown is active until %dms (%dms too early, released by coder %d)", log.LineNo, coder, d.side, d.id, ts, cooldownUntil[d.id], diff, lastReleasedBy[d.id]),
							Advice:      "Dongles must cool down for dongle_cooldown ms after being released before another coder can start compiling with them.",
						}
					}
					warnings = append(warnings, fmt.Sprintf("Line %d: coder %d compiled using %s dongle %d %dms before cooldown expiration (active until %dms, accepted within OS scheduling jitter <=10ms)", log.LineNo, coder, d.side, d.id, diff, cooldownUntil[d.id]))
				}
			}

			compilingCoders[coder] = true
			compileCounts[coder]++
			lastCompile[coder] = ts
			lastDebug[coder] = nil
			lastRefactor[coder] = nil

		} else if state == "is debugging" {
			delete(compilingCoders, coder)

			// Set dongle cooldown
			if n > 1 {
				left := coder - 1
				if coder == 1 {
					left = n
				}
				right := coder
				cooldownUntil[left] = ts + params.DongleCooldown
				cooldownUntil[right] = ts + params.DongleCooldown
				lastReleasedBy[left] = coder
				lastReleasedBy[right] = coder
			}

			// Check compile duration
			elapsed := ts - lastCompile[coder]
			if elapsed < params.TimeToCompile {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Compile phase ended too early on line %d: coder %d compiled for only %dms (expected >= %dms)", log.LineNo, coder, elapsed, params.TimeToCompile),
					Advice:      "Check your precise timing implementation (usleep loop or gettimeofday diff) for compile phase.",
				}
			}
			if elapsed > (params.TimeToCompile*3/2 + 20) {
				drift := elapsed - params.TimeToCompile
				warnings = append(warnings, fmt.Sprintf("Line %d: coder %d compile duration (%dms) drifted +%dms above expected %dms", log.LineNo, coder, elapsed, drift, params.TimeToCompile))
			}
			dbgTS := ts
			lastDebug[coder] = &dbgTS

		} else if state == "is refactoring" {
			if lastDebug[coder] == nil {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Invalid sequence on line %d: coder %d started refactoring without debugging first", log.LineNo, coder),
					Advice:      "A coder must complete debugging before starting refactoring.",
				}
			}
			elapsed := ts - *lastDebug[coder]
			if elapsed < params.TimeToDebug {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Debug phase ended too early on line %d: coder %d debugged for only %dms (expected >= %dms)", log.LineNo, coder, elapsed, params.TimeToDebug),
					Advice:      "Ensure time_to_debug is fully respected before entering refactor state.",
				}
			}
			if elapsed > (params.TimeToDebug*3/2 + 20) {
				drift := elapsed - params.TimeToDebug
				warnings = append(warnings, fmt.Sprintf("Line %d: coder %d debug duration (%dms) drifted +%dms above expected %dms", log.LineNo, coder, elapsed, drift, params.TimeToDebug))
			}
			refTS := ts
			lastRefactor[coder] = &refTS

		} else if state == "burned out" {
			delete(compilingCoders, coder)
			burnouts++

			deadline := lastCompile[coder] + params.TimeToBurnout
			lateness := ts - deadline

			if lateness < 0 {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Premature burnout on line %d: coder %d burned out at %dms, before their deadline %dms (%dms early)", log.LineNo, coder, ts, deadline, -lateness),
					Advice:      "A coder only burns out if they haven't compiled within time_to_burnout.",
				}
			}
			if lateness > 25 {
				return &ValidationResult{
					Status:      model.StatusFailed,
					Explanation: fmt.Sprintf("Burnout detection delayed on line %d: coder %d deadline was %dms, but burnout announced at %dms (%dms delay, tolerance is <=25ms)", log.LineNo, coder, deadline, ts, lateness),
					Advice:      "Your monitoring thread / reaper loop must detect starvation within 10ms (maximum 25ms under system load).",
				}
			}
			if lateness > 10 {
				warnings = append(warnings, fmt.Sprintf("Line %d: coder %d burnout announced with %dms delay (spec: <=10ms, accepted within tolerance <=25ms)", log.LineNo, coder, lateness))
			}

			// Verify NO subsequent simulation actions logged with timestamp > burnout ts
			for _, sub := range logs[idx+1:] {
				if sub.TS > ts {
					return &ValidationResult{
						Status:      model.StatusFailed,
						Explanation: fmt.Sprintf("Post-mortem logging violation: coder %d burned out at %dms on line %d, but subsequent action '%s' was logged at %dms on line %d", coder, ts, log.LineNo, sub.Raw, sub.TS, sub.LineNo),
						Advice:      "As soon as a coder burns out, the simulation must stop immediately and no further actions can be printed.",
					}
				}
			}
		}

		lastState[coder] = state
	}

	// Outcome evaluation
	if burnouts > 1 {
		return &ValidationResult{
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Multiple burnout events logged (%d burnouts)", burnouts),
			Advice:      "The simulation must terminate as soon as the first coder burns out.",
		}
	}
	if expectNoBurnout && burnouts > 0 {
		return &ValidationResult{
			Status:      model.StatusFailed,
			Explanation: "A coder burned out, but this scenario specifies that no coder should starve",
			Advice:      "Check deadlock prevention, fair dongle sharing, and scheduler priority logic.",
		}
	}
	if expectBurnout && burnouts == 0 {
		return &ValidationResult{
			Status:      model.StatusFailed,
			Explanation: "Expected coder burnout/starvation did not occur",
			Advice:      "In this scenario, timing constraints make starvation inevitable. Verify that your monitoring thread terminates the program.",
		}
	}

	// Verify required compiles reached when no burnout
	if burnouts == 0 && params.NumCompiles > 0 {
		var notDone []string
		for i := 1; i <= n; i++ {
			if compileCounts[i] < params.NumCompiles {
				notDone = append(notDone, fmt.Sprintf("coder %d (%d/%d compiles)", i, compileCounts[i], params.NumCompiles))
			}
		}
		if len(notDone) > 0 {
			limit := 5
			if len(notDone) < limit {
				limit = len(notDone)
			}
			return &ValidationResult{
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Simulation terminated before all coders reached required compiles: %s", strings.Join(notDone[:limit], ", ")),
				Advice:      "Ensure the simulation keeps running until every single coder has completed number_of_compiles_required.",
			}
		}
	}

	return &ValidationResult{
		Status:   model.StatusPassed,
		Warnings: warnings,
	}
}

// ValidateErrorCase checks if an invalid CLI invocation is cleanly handled.
func ValidateErrorCase(returnCode int, stdout, stderr string) *ValidationResult {
	merged := stdout + "\n" + stderr
	lines := strings.Split(merged, "\n")
	errorKeywords := []string{"error", "invalid", "usage", "incorrect", "bad", "wrong"}
	hasErrorWord := false

	for _, l := range lines {
		low := strings.ToLower(strings.TrimSpace(l))
		for _, kw := range errorKeywords {
			if strings.Contains(low, kw) {
				hasErrorWord = true
				break
			}
		}
		if hasErrorWord {
			break
		}
	}

	if returnCode != 0 {
		return &ValidationResult{
			Status: model.StatusPassed,
		}
	}
	if hasErrorWord {
		return &ValidationResult{
			Status:   model.StatusPassed,
			Warnings: []string{"Invalid arguments rejected with error message, but process exited with returncode 0 (expected non-zero)"},
		}
	}

	return &ValidationResult{
		Status:      model.StatusFailed,
		Explanation: fmt.Sprintf("Expected invalid argument rejection, but binary exited successfully with returncode 0 without error message (stdout='%s', stderr='%s')", strings.TrimSpace(stdout), strings.TrimSpace(stderr)),
		Advice:      "Validate argument counts, numeric syntax, positive limits, and scheduler names, and return non-zero code on error.",
	}
}
