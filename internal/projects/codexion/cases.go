package codexion

import (
	"time"

	"github.com/tristan-gscn/repaco/internal/model"
)

type CaseKind int

const (
	KindSyntaxError CaseKind = iota
	KindLogicError
	KindNominal
	KindBurnout
	KindStress
	KindMemory
)

// TestCaseDef defines a declarative codexion test case.
type TestCaseDef struct {
	ID              string
	File            string
	Name            string
	Category        model.Category
	Description     string
	Kind            CaseKind
	Args            []string
	Timeout         time.Duration
	ExpectBurnout   bool
	ExpectNoBurnout bool
	ExpectError     bool
}

// GetDefaultCases returns the full battery of declarative tests for Codexion.
// Beginners: to add a new test case, simply add an entry to one of the slices below!
func GetDefaultCases() []TestCaseDef {
	var defs []TestCaseDef

	// =========================================================================
	// 1. Argument Syntax Cases (codexion_syntax.c)
	// Non-numeric inputs, float parameters, huge overflows, invalid schedulers.
	// =========================================================================
	syntaxCases := []struct {
		id, name, desc string
		args           []string
	}{
		{"syntax_coder_alpha", "Non-numeric coders count ('banana')", "Invalid number_of_coders string must be rejected", []string{"banana", "200", "300", "400", "500", "5", "10", "fifo"}},
		{"syntax_burnout_alpha", "Non-numeric burnout time ('banana')", "Invalid time_to_burnout string must be rejected", []string{"10", "banana", "300", "400", "500", "5", "10", "fifo"}},
		{"syntax_compile_alpha", "Non-numeric compile time ('banana')", "Invalid time_to_compile string must be rejected", []string{"10", "200", "banana", "400", "500", "5", "10", "fifo"}},
		{"syntax_debug_alpha", "Non-numeric debug time ('banana')", "Invalid time_to_debug string must be rejected", []string{"10", "200", "300", "banana", "500", "5", "10", "fifo"}},
		{"syntax_refactor_alpha", "Non-numeric refactor time ('banana')", "Invalid time_to_refactor string must be rejected", []string{"10", "200", "300", "400", "banana", "5", "10", "fifo"}},
		{"syntax_compiles_alpha", "Non-numeric compiles count ('banana')", "Invalid number_of_compiles string must be rejected", []string{"10", "200", "300", "400", "500", "banana", "10", "fifo"}},
		{"syntax_cooldown_alpha", "Non-numeric dongle cooldown ('banana')", "Invalid dongle_cooldown string must be rejected", []string{"10", "200", "300", "400", "500", "5", "banana", "fifo"}},
		{"syntax_scheduler_invalid", "Invalid scheduler name ('banana')", "Invalid scheduler string must be rejected", []string{"10", "200", "300", "400", "500", "5", "10", "banana"}},
		{"syntax_float_arg", "Floating point argument ('800.5')", "Floating point argument must be safely rejected", []string{"4", "800.5", "200", "200", "200", "5", "10", "fifo"}},
		{"syntax_overflow", "Integer overflow (999999999999999999999999999999)", "Number exceeding integer limits must be safely rejected", []string{"4", "999999999999999999999999999999", "200", "200", "200", "5", "10", "fifo"}},
	}
	for _, sc := range syntaxCases {
		defs = append(defs, TestCaseDef{
			ID:          "codexion_" + sc.id,
			File:        "codexion_syntax.c",
			Name:        sc.name,
			Category:    model.CategoryEdgeCase,
			Description: sc.desc,
			Kind:        KindSyntaxError,
			Args:        sc.args,
			Timeout:     3 * time.Second,
			ExpectError: true,
		})
	}

	// =========================================================================
	// 2. Argument Logic Cases (codexion_logic.c)
	// Negative values, 0 coders, case sensitivity, spacing, missing args.
	// =========================================================================
	logicCases := []struct {
		id, name, desc string
		args           []string
	}{
		{"logic_negative_duration", "Negative duration parameter (-400ms)", "Negative duration must be rejected", []string{"10", "200", "300", "-400", "500", "5", "10", "edf"}},
		{"logic_too_many_args", "Too many arguments provided (9 args)", "Command line with extra arguments must be rejected", []string{"too", "10", "200", "300", "400", "500", "5", "10", "edf"}},
		{"logic_zero_coders", "Zero coders count ('0')", "0 coders must be rejected", []string{"0", "800", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_negative_coders", "Negative coders count ('-5')", "Negative number of coders must be rejected", []string{"-5", "800", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_caps_scheduler", "Uppercase scheduler name ('FIFO')", "Uppercase scheduler must be rejected (subject requires 'fifo')", []string{"4", "800", "200", "200", "200", "5", "10", "FIFO"}},
		{"logic_empty_arg", "Empty string argument ('')", "Empty string parameter must be rejected", []string{"4", "", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_leading_spaces", "Leading spaces in argument (' 200')", "Leading space in number parameter must be rejected", []string{"4", "800", " 200", "200", "200", "5", "10", "fifo"}},
		{"logic_trailing_tab", "Trailing tab in argument ('200\\t')", "Trailing tab in number parameter must be rejected", []string{"4", "800", "200\t", "200", "200", "5", "10", "fifo"}},
		{"logic_bad_scheduler", "Unsupported scheduler ('round_robin')", "Unsupported scheduler name must be rejected", []string{"4", "800", "200", "200", "200", "5", "10", "round_robin"}},
		{"logic_missing_args", "Missing required scheduler argument", "Command line missing 8th argument must be rejected", []string{"10", "200", "300", "400", "500", "5", "10"}},
		{"logic_no_args", "No arguments provided at all", "Invoking binary with 0 arguments must display usage and exit", []string{}},
	}
	for _, lc := range logicCases {
		defs = append(defs, TestCaseDef{
			ID:          "codexion_" + lc.id,
			File:        "codexion_logic.c",
			Name:        lc.name,
			Category:    model.CategoryEdgeCase,
			Description: lc.desc,
			Kind:        KindLogicError,
			Args:        lc.args,
			Timeout:     3 * time.Second,
			ExpectError: true,
		})
	}

	// =========================================================================
	// 3. Nominal Simulation Cases (codexion_nominal.c)
	// Valid runs where NO burnout must occur.
	// =========================================================================
	nominalCases := []struct {
		id, name, desc string
		args           []string
		timeout        time.Duration
	}{
		{"nom_1_basic_fifo", "4 coders, FIFO scheduler (basic nominal)", "4 coders, balanced cycles under FIFO scheduler", []string{"4", "800", "200", "200", "200", "5", "10", "fifo"}, 6 * time.Second},
		{"nom_2_basic_edf", "4 coders, EDF scheduler (basic nominal)", "4 coders, balanced cycles under EDF scheduler", []string{"4", "800", "200", "200", "200", "5", "10", "edf"}, 6 * time.Second},
		{"nom_3_success_fifo", "10 coders, 5 compiles, FIFO", "10 coders, 5 compiles required each, verifies clean termination", []string{"10", "10000", "100", "100", "100", "5", "50", "fifo"}, 8 * time.Second},
		{"nom_4_large_edf", "20 coders, 500ms actions, EDF", "20 coders with 500ms actions under EDF scheduler", []string{"20", "5000", "500", "500", "500", "10", "100", "edf"}, 12 * time.Second},
		{"nom_5_low_cooldown", "5 coders, 1ms dongle cooldown", "5 coders with minimal 1ms cooldown to test fast handovers", []string{"5", "2000", "100", "100", "100", "20", "1", "fifo"}, 8 * time.Second},
		{"nom_6_long_actions", "3 coders, 600ms long action phases", "3 coders with long action phases (600ms each)", []string{"3", "4000", "600", "600", "600", "2", "50", "fifo"}, 8 * time.Second},
		{"nom_7_zero_compile", "0 compiles required: instant clean exit", "0 compiles required: coders must not compile and cleanly exit", []string{"5", "1000", "200", "200", "200", "0", "10", "fifo"}, 3 * time.Second},
		{"nom_8_tight_timings", "Tight margins (410ms burnout vs 300ms cycle)", "Tight timing margins under EDF: tests timer precision", []string{"4", "410", "100", "100", "100", "5", "5", "edf"}, 6 * time.Second},
	}
	for _, nc := range nominalCases {
		defs = append(defs, TestCaseDef{
			ID:              "codexion_" + nc.id,
			File:            "codexion_nominal.c",
			Name:            nc.name,
			Category:        model.CategoryMandatory,
			Description:     nc.desc,
			Kind:            KindNominal,
			Args:            nc.args,
			Timeout:         nc.timeout,
			ExpectNoBurnout: true,
		})
	}

	// =========================================================================
	// 4. Burnout & Starvation Cases (codexion_burnout.c)
	// Infeasible scenarios where a coder MUST burn out within accurate timing.
	// =========================================================================
	burnoutCases := []struct {
		id, name, desc string
		args           []string
		timeout        time.Duration
	}{
		{"burn_starvation_fifo", "Neighbor starvation (FIFO)", "3 coders with 600ms compile starves neighbors, burnout expected (FIFO)", []string{"3", "1000", "600", "10", "10", "5", "100", "fifo"}, 5 * time.Second},
		{"burn_starvation_edf", "Neighbor starvation (EDF)", "3 coders with 600ms compile starves neighbors, burnout expected (EDF)", []string{"3", "1000", "600", "10", "10", "5", "100", "edf"}, 5 * time.Second},
		{"burn_one_coder_fifo", "1 single coder on table (FIFO)", "1 coder cannot acquire 2 dongles; must starve and burn out (FIFO)", []string{"1", "1000", "200", "200", "200", "5", "50", "fifo"}, 4 * time.Second},
		{"burn_one_coder_edf", "1 single coder on table (EDF)", "1 coder cannot acquire 2 dongles; must starve and burn out (EDF)", []string{"1", "1000", "200", "200", "200", "5", "50", "edf"}, 4 * time.Second},
		{"burn_immediate", "Immediate burnout (1ms burnout time)", "1ms burnout: coder must burn out immediately (<=10ms delay)", []string{"2", "1", "200", "200", "200", "5", "10", "fifo"}, 3 * time.Second},
		{"burn_cooldown_hell", "High cooldown starvation (2000ms cooldown)", "2000ms cooldown: dongles unavailable in time, burnout expected", []string{"2", "1000", "100", "100", "100", "5", "2000", "fifo"}, 5 * time.Second},
	}
	for _, bc := range burnoutCases {
		defs = append(defs, TestCaseDef{
			ID:            "codexion_" + bc.id,
			File:          "codexion_burnout.c",
			Name:          bc.name,
			Category:      model.CategoryMandatory,
			Description:   bc.desc,
			Kind:          KindBurnout,
			Args:          bc.args,
			Timeout:       bc.timeout,
			ExpectBurnout: true,
		})
	}

	// =========================================================================
	// 5. Concurrency & Contention Stress Cases (codexion_stress.c)
	// Extreme thread counts (100, 300, 999 threads) to expose race conditions.
	// =========================================================================
	stressCases := []struct {
		id, name, desc  string
		args            []string
		timeout         time.Duration
		expectNoBurnout bool
	}{
		{"stress_100_coders", "100 simultaneous coders concurrency stress", "Concurrency stress test with 100 simultaneous threads", []string{"100", "10000", "66", "24", "87", "10", "10", "fifo"}, 10 * time.Second, true},
		{"stress_300_coders", "300 coders scalability under EDF", "High thread count scalability under EDF scheduler", []string{"300", "10000", "100", "100", "100", "5", "10", "edf"}, 12 * time.Second, true},
		{"stress_999_threads", "999 threads extreme contention test", "Extreme thread contention (999 coders) to catch data races", []string{"999", "1000", "200", "200", "200", "5", "50", "fifo"}, 10 * time.Second, false},
	}
	for _, st := range stressCases {
		defs = append(defs, TestCaseDef{
			ID:              "codexion_" + st.id,
			File:            "codexion_stress.c",
			Name:            st.name,
			Category:        model.CategoryBenchmark,
			Description:     st.desc,
			Kind:            KindStress,
			Args:            st.args,
			Timeout:         st.timeout,
			ExpectNoBurnout: st.expectNoBurnout,
		})
	}

	// =========================================================================
	// 6. Memory & Data Race Safety (codexion_memory.c)
	// Verifies zero memory leaks via macOS leaks or Valgrind.
	// =========================================================================
	defs = append(defs, TestCaseDef{
		ID:              "codexion_memory_leak",
		File:            "codexion_memory.c",
		Name:            "Memory leak check (macOS leaks)",
		Category:        model.CategoryMemory,
		Description:     "Runs 4 coders simulation and verifies zero memory leaks on completion",
		Kind:            KindMemory,
		Args:            []string{"4", "800", "100", "100", "100", "2", "10", "fifo"},
		Timeout:         8 * time.Second,
		ExpectNoBurnout: true,
	})

	return defs
}
