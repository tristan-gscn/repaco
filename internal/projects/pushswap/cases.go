package pushswap

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/tristan-gscn/repaco/internal/model"
)

type CaseType int

const (
	CaseError CaseType = iota
	CaseIdentity
	CaseSort
	CaseBenchmark
	CaseMemory
	CaseBonus
)

// TestCaseDef defines a declarative push_swap test case.
type TestCaseDef struct {
	ID          string
	File        string
	Name        string
	Category    model.Category
	Description string
	Type        CaseType
	Args        []string
	MaxOps      int
	Timeout     time.Duration
}

// GetDefaultCases returns the full declarative test suite for push_swap.
// Beginners: to add a new test case, simply add an entry to one of the slices below!
func GetDefaultCases(hasBonus bool) []TestCaseDef {
	var definitions []TestCaseDef

	// =========================================================================
	// 1. Error Management (push_swap_errors.c)
	// Must write "Error\n" to stderr and exit cleanly.
	// =========================================================================
	errorDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"err_alpha", "Non-numeric argument in list", "Arguments containing letters must write 'Error\\n' to stderr", []string{"1", "2", "3", "a", "5"}},
		{"err_alpha_only", "Single non-numeric argument", "Word argument 'abc' must write 'Error\\n' to stderr", []string{"abc"}},
		{"err_overflow", "Integer overflow (> INT_MAX)", "Number exceeding INT_MAX (2147483648) must write 'Error\\n'", []string{"2147483648"}},
		{"err_underflow", "Integer underflow (< INT_MIN)", "Number below INT_MIN (-2147483649) must write 'Error\\n'", []string{"-2147483649"}},
		{"err_duplicates", "Duplicate numbers", "Duplicate numbers in list must write 'Error\\n'", []string{"1", "2", "3", "2", "4"}},
		{"err_empty_string", "Empty string parameter", "Empty string '' must write 'Error\\n'", []string{""}},
		{"err_spaces_string", "Spaces-only string parameter", "String with only spaces '   ' must write 'Error\\n'", []string{"   "}},
		{"err_isolated_plus", "Isolated '+' sign", "Isolated sign '+' must write 'Error\\n'", []string{"+"}},
		{"err_isolated_minus", "Isolated '-' sign", "Isolated sign '-' must write 'Error\\n'", []string{"-"}},
		{"err_double_sign", "Double sign '++1'", "Invalid double sign '++1' must write 'Error\\n'", []string{"++1"}},
		{"err_invalid_suffix", "Alphanumeric suffix '123a'", "Number with alphanumeric suffix must write 'Error\\n'", []string{"123a"}},
		{"err_middle_empty", "Empty parameter in middle", "Empty string in middle of valid numbers must write 'Error\\n'", []string{"1", "", "2"}},
	}
	for _, ed := range errorDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + ed.id,
			File:        "push_swap_errors.c",
			Name:        ed.name,
			Category:    model.CategoryEdgeCase,
			Description: ed.desc,
			Type:        CaseError,
			Args:        ed.args,
			Timeout:     2 * time.Second,
		})
	}

	// =========================================================================
	// 2. Identity / Already Sorted (push_swap_identity.c)
	// Must produce zero instructions and exit cleanly with code 0.
	// =========================================================================
	identityDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"id_no_args", "No arguments", "Running push_swap with 0 parameters must display nothing", []string{}},
		{"id_single_arg", "Single positive number", "Running push_swap with single number (42) must display nothing", []string{"42"}},
		{"id_single_negative", "Single negative number", "Running push_swap with single negative number (-42) must display nothing", []string{"-42"}},
		{"id_sorted_2", "Already sorted 2 numbers", "Running push_swap with 1 2 must display nothing", []string{"1", "2"}},
		{"id_sorted_3", "Already sorted 3 numbers", "Running push_swap with 1 2 3 must display nothing", []string{"1", "2", "3"}},
		{"id_sorted_5", "Already sorted 5 numbers", "Running push_swap with 1 2 3 4 5 must display nothing", []string{"1", "2", "3", "4", "5"}},
		{"id_sorted_quoted", "Already sorted quoted string", "Running push_swap with '1 2 3 4 5' must display nothing", []string{"1 2 3 4 5"}},
		{"id_sorted_negatives", "Already sorted negative numbers", "Running push_swap with '-10 -5 0 5 10' must display nothing", []string{"-10", "-5", "0", "5", "10"}},
	}
	for _, id := range identityDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + id.id,
			File:        "push_swap_identity.c",
			Name:        id.name,
			Category:    model.CategoryMandatory,
			Description: id.desc,
			Type:        CaseIdentity,
			Args:        id.args,
			Timeout:     2 * time.Second,
		})
	}

	// =========================================================================
	// 3. Small Sort (push_swap_small.c)
	// Mandatory small stack permutations (2, 3, 4, 5 numbers).
	// =========================================================================
	smallDefs := []struct {
		id, name, desc string
		args           []string
		maxOps         int
	}{
		{"small_2_rev", "Sort 2 numbers (reverse)", "Sorts '2 1' in <= 1 operations", []string{"2", "1"}, 1},
		{"small_3_perm_213", "Sort 3 numbers (2 1 3)", "Sorts '2 1 3' in <= 3 operations", []string{"2", "1", "3"}, 3},
		{"small_3_perm_321", "Sort 3 numbers (3 2 1)", "Sorts '3 2 1' in <= 3 operations", []string{"3", "2", "1"}, 3},
		{"small_3_perm_312", "Sort 3 numbers (3 1 2)", "Sorts '3 1 2' in <= 3 operations", []string{"3", "1", "2"}, 3},
		{"small_3_perm_132", "Sort 3 numbers (1 3 2)", "Sorts '1 3 2' in <= 3 operations", []string{"1", "3", "2"}, 3},
		{"small_3_perm_231", "Sort 3 numbers (2 3 1)", "Sorts '2 3 1' in <= 3 operations", []string{"2", "3", "1"}, 3},
		{"small_4_reverse", "Sort 4 numbers (4 3 2 1)", "Sorts '4 3 2 1' in <= 12 operations", []string{"4", "3", "2", "1"}, 12},
		{"small_4_random", "Sort 4 numbers (2 4 1 3)", "Sorts '2 4 1 3' in <= 12 operations", []string{"2", "4", "1", "3"}, 12},
		{"small_5_reverse", "Sort 5 numbers (5 4 3 2 1)", "Sorts '5 4 3 2 1' in <= 12 operations", []string{"5", "4", "3", "2", "1"}, 12},
		{"small_5_perm1", "Sort 5 numbers (1 5 2 4 3)", "Sorts '1 5 2 4 3' in <= 12 operations", []string{"1", "5", "2", "4", "3"}, 12},
		{"small_5_perm2", "Sort 5 numbers (2 3 5 1 4)", "Sorts '2 3 5 1 4' in <= 12 operations", []string{"2", "3", "5", "1", "4"}, 12},
		{"small_5_perm3", "Sort 5 numbers (4 1 5 3 2)", "Sorts '4 1 5 3 2' in <= 12 operations", []string{"4", "1", "5", "3", "2"}, 12},
		{"small_5_quoted", "Sort 5 numbers quoted ('5 2 4 1 3')", "Sorts space-separated string in <= 12 operations", []string{"5 2 4 1 3"}, 12},
		{"small_5_negatives", "Sort 5 numbers with negatives", "Sorts '-2 5 -10 0 3' in <= 12 operations", []string{"-2", "5", "-10", "0", "3"}, 12},
	}
	for _, sd := range smallDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + sd.id,
			File:        "push_swap_small.c",
			Name:        sd.name,
			Category:    model.CategoryMandatory,
			Description: sd.desc,
			Type:        CaseSort,
			Args:        sd.args,
			MaxOps:      sd.maxOps,
			Timeout:     2 * time.Second,
		})
	}

	// =========================================================================
	// 4. Benchmarks (push_swap_benchmarks.c)
	// Stacks of 100 (< 700 ops for 5/5) and 500 numbers (< 5500 ops for 5/5).
	// =========================================================================
	bench100_1 := generatePermutationArgs(100, 42)
	bench100_2 := generatePermutationArgs(100, 1337)
	bench100_3 := generatePermutationArgs(100, 2026)
	bench100_rev := generateReverseArgs(100)

	bench500_1 := generatePermutationArgs(500, 42)
	bench500_2 := generatePermutationArgs(500, 1337)
	bench500_rev := generateReverseArgs(500)

	benchDefs := []struct {
		id, name, desc string
		args           []string
		maxOps         int
		timeout        time.Duration
	}{
		{"bench_100_set1", "100 numbers (seed 42)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_1, 700, 5 * time.Second},
		{"bench_100_set2", "100 numbers (seed 1337)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_2, 700, 5 * time.Second},
		{"bench_100_set3", "100 numbers (seed 2026)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_3, 700, 5 * time.Second},
		{"bench_100_reverse", "100 numbers (descending 100..1)", "Sorts 100 reverse numbers, target < 700 ops for 5/5", bench100_rev, 700, 5 * time.Second},
		{"bench_500_set1", "500 numbers (seed 42)", "Sorts 500 random numbers, target < 5500 ops for 5/5", bench500_1, 5500, 10 * time.Second},
		{"bench_500_set2", "500 numbers (seed 1337)", "Sorts 500 random numbers, target < 5500 ops for 5/5", bench500_2, 5500, 10 * time.Second},
		{"bench_500_reverse", "500 numbers (descending 500..1)", "Sorts 500 reverse numbers, target < 5500 ops for 5/5", bench500_rev, 5500, 10 * time.Second},
	}
	for _, bd := range benchDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + bd.id,
			File:        "push_swap_benchmarks.c",
			Name:        bd.name,
			Category:    model.CategoryBenchmark,
			Description: bd.desc,
			Type:        CaseBenchmark,
			Args:        bd.args,
			MaxOps:      bd.maxOps,
			Timeout:     bd.timeout,
		})
	}

	// =========================================================================
	// 5. Memory Checks (push_swap_memory.c)
	// Verifies zero memory leaks via Valgrind or macOS leaks.
	// =========================================================================
	memDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"mem_small", "Memory check: small sort", "Verifies 0 memory leaks on sorting 5 numbers", []string{"5", "4", "3", "2", "1"}},
		{"mem_error", "Memory check: error exit", "Verifies 0 memory leaks when exiting on duplicate error", []string{"1", "2", "2"}},
		{"mem_100", "Memory check: 100 numbers", "Verifies 0 memory leaks after sorting 100 numbers", bench100_1},
	}
	for _, md := range memDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + md.id,
			File:        "push_swap_memory.c",
			Name:        md.name,
			Category:    model.CategoryMemory,
			Description: md.desc,
			Type:        CaseMemory,
			Args:        md.args,
			Timeout:     5 * time.Second,
		})
	}

	// =========================================================================
	// 6. Bonus Checker (push_swap_bonus.c)
	// Verifies checker binary behavior if present in project.
	// =========================================================================
	if hasBonus {
		bonusDefs := []struct {
			id, name, desc string
		}{
			{"bonus_valid_sort", "Checker: valid sort output", "Piping push_swap instructions to checker displays 'OK\\n'"},
			{"bonus_ko", "Checker: incorrect instructions", "Piping unsorted instructions to checker displays 'KO\\n'"},
			{"bonus_invalid_op", "Checker: invalid instruction 'foo'", "Checker displays 'Error\\n' on invalid instruction"},
			{"bonus_error_args", "Checker: invalid arguments", "Checker displays 'Error\\n' on non-numeric arguments"},
		}
		for _, bnd := range bonusDefs {
			definitions = append(definitions, TestCaseDef{
				ID:          "pushswap_" + bnd.id,
				File:        "push_swap_bonus.c",
				Name:        bnd.name,
				Category:    model.CategoryBonus,
				Description: bnd.desc,
				Type:        CaseBonus,
				Timeout:     3 * time.Second,
			})
		}
	}

	return definitions
}

func generatePermutationArgs(count int, seed int64) []string {
	r := rand.New(rand.NewSource(seed))
	vals := make([]int, count)
	for i := 0; i < count; i++ {
		vals[i] = i - count/2
	}
	r.Shuffle(count, func(i, j int) {
		vals[i], vals[j] = vals[j], vals[i]
	})
	res := make([]string, count)
	for i, v := range vals {
		res[i] = strconv.Itoa(v)
	}
	return res
}

func generateReverseArgs(count int) []string {
	res := make([]string, count)
	for i := 0; i < count; i++ {
		res[i] = strconv.Itoa(count - i)
	}
	return res
}
