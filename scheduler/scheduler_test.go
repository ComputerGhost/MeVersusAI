package scheduler

import (
	"flag"
	"os"
	"testing"
)

var factory func() Scheduler
var target = flag.String("target", "", "which implementation (ai|human) to test")

func TestMain(m *testing.M) {
	flag.Parse()

	switch *target {
	case "ai":
		factory = func() Scheduler { return NewAIScheduler() }
	case "human":
		factory = func() Scheduler { return NewHumanScheduler() }
	default:
		factory = func() Scheduler { return NewHumanScheduler() }
		//panic("invalid target: " + *target)
	}

	os.Exit(m.Run())
}

func TestOrderEmptyScheduler(t *testing.T) {
	s := factory()

	got, err := s.Order()
	if err != nil {
		t.Fatalf("Order() returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Order() = %v, want empty slice", got)
	}
}

func TestOrderSingleTask(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")

	assertOrder(t, s, []string{"compile"})
}

func TestOrderLinearDependencies(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile")
	mustAdd(t, s, "deploy", "test")

	assertOrder(t, s, []string{"compile", "test", "deploy"})
}

func TestOrderBranchingDependenciesUseLexicographicTieBreak(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile")
	mustAdd(t, s, "package", "compile")
	mustAdd(t, s, "deploy", "package", "test")

	assertOrder(t, s, []string{"compile", "package", "test", "deploy"})
}

func TestOrderIndependentTasksUseLexicographicTieBreak(t *testing.T) {
	s := factory()

	mustAdd(t, s, "zeta")
	mustAdd(t, s, "alpha")
	mustAdd(t, s, "middle")

	assertOrder(t, s, []string{"alpha", "middle", "zeta"})
}

// AI wrote this test, but it seems to be for a depth-first algorithm, which is
// not the algorithm suggested in the instructions. When confronted about this,
// it agreed that it was wrong and generated TestOrderComplexGraph. When told
// that this new function loses the deterministic testing, it then generated
// this function again and said it solves both problems. It does not.
func _TestOrderComplexGraphIsDeterministic(t *testing.T) {
	s := factory()

	mustAdd(t, s, "lint")
	mustAdd(t, s, "compile")
	mustAdd(t, s, "unit", "compile")
	mustAdd(t, s, "integration", "compile")
	mustAdd(t, s, "package", "integration", "unit")
	mustAdd(t, s, "security-scan", "package")
	mustAdd(t, s, "deploy", "lint", "package", "security-scan")

	assertOrder(t, s, []string{
		"compile",
		"integration",
		"lint",
		"unit",
		"package",
		"security-scan",
		"deploy",
	})
}

func TestOrderComplexGraph(t *testing.T) {
	s := factory()

	mustAdd(t, s, "lint")
	mustAdd(t, s, "compile")
	mustAdd(t, s, "unit", "compile")
	mustAdd(t, s, "integration", "compile")
	mustAdd(t, s, "package", "integration", "unit")
	mustAdd(t, s, "security-scan", "package")
	mustAdd(t, s, "deploy", "lint", "package", "security-scan")

	order, err := s.Order()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContainsAll(t, order, []string{
		"lint",
		"compile",
		"unit",
		"integration",
		"package",
		"security-scan",
		"deploy",
	})

	assertBefore(t, order, "compile", "unit")
	assertBefore(t, order, "compile", "integration")
	assertBefore(t, order, "unit", "package")
	assertBefore(t, order, "integration", "package")
	assertBefore(t, order, "package", "security-scan")
	assertBefore(t, order, "lint", "deploy")
	assertBefore(t, order, "package", "deploy")
	assertBefore(t, order, "security-scan", "deploy")
}

func TestAddTaskRejectsEmptyTaskName(t *testing.T) {
	s := factory()

	if err := s.AddTask(""); err == nil {
		t.Fatal("AddTask(\"\") returned nil error, want non-nil error")
	}

	assertOrder(t, s, []string{})
}

func TestAddTaskRejectsEmptyDependencyName(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")

	if err := s.AddTask("test", "compile", ""); err == nil {
		t.Fatal("AddTask with empty dependency returned nil error, want non-nil error")
	}

	assertOrder(t, s, []string{"compile"})
}

func TestAddTaskRejectsMissingDependency(t *testing.T) {
	s := factory()

	if err := s.AddTask("test", "compile"); err == nil {
		t.Fatal("AddTask with missing dependency returned nil error, want non-nil error")
	}

	assertOrder(t, s, []string{})
}

func TestAddTaskRejectsSelfDependency(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")

	if err := s.AddTask("compile", "compile"); err == nil {
		t.Fatal("AddTask with self dependency returned nil error, want non-nil error")
	}

	assertOrder(t, s, []string{"compile"})
}

func TestAddTaskDeduplicatesDependencies(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile", "compile", "compile")

	assertOrder(t, s, []string{"compile", "test"})
}

func TestAddTaskReplacesExistingTaskDependencies(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "lint")
	mustAdd(t, s, "test", "compile")
	mustAdd(t, s, "test", "lint")

	order, err := s.Order()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContainsAll(t, order, []string{"compile", "lint", "test"})
	assertBefore(t, order, "compile", "test")
	assertBefore(t, order, "lint", "test")
}

func TestAddTaskValidationFailureLeavesExistingTaskUnchanged(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile")

	if err := s.AddTask("test", "missing"); err == nil {
		t.Fatal("AddTask replacing with missing dependency returned nil error, want non-nil error")
	}

	assertOrder(t, s, []string{"compile", "test"})
}

func TestRemoveTaskRemovesExistingTask(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "lint")

	s.RemoveTask("compile")

	assertOrder(t, s, []string{"lint"})
}

func TestRemoveTaskMissingTaskIsNoOp(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	s.RemoveTask("missing")

	assertOrder(t, s, []string{"compile"})
}

func TestOrderErrorsWhenRemovedTaskIsStillADependency(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile")

	s.RemoveTask("compile")

	if got, err := s.Order(); err == nil {
		t.Fatalf("Order() = %v, nil error; want missing dependency error", got)
	}
}

func TestOrderErrorsOnCycleCreatedByReplacement(t *testing.T) {
	s := factory()

	mustAdd(t, s, "a")
	mustAdd(t, s, "b", "a")
	mustAdd(t, s, "c", "b")
	mustAdd(t, s, "a", "c")

	if got, err := s.Order(); err == nil {
		t.Fatalf("Order() = %v, nil error; want cycle error", got)
	}
}

func TestOrderDoesNotMutateScheduler(t *testing.T) {
	s := factory()

	mustAdd(t, s, "compile")
	mustAdd(t, s, "test", "compile")
	mustAdd(t, s, "package", "compile")

	want := []string{"compile", "package", "test"}
	assertOrder(t, s, want)
	assertOrder(t, s, want)
}

func mustAdd(t *testing.T, s Scheduler, name string, dependencies ...string) {
	t.Helper()

	if err := s.AddTask(name, dependencies...); err != nil {
		t.Fatalf("AddTask(%q, %v) returned unexpected error: %v", name, dependencies, err)
	}
}

func assertOrder(t *testing.T, s Scheduler, want []string) {
	t.Helper()

	got, err := s.Order()
	if err != nil {
		t.Fatalf("Order() returned unexpected error: %v", err)
	}

	if !sameStrings(got, want) {
		t.Fatalf("Order() = %v, want %v", got, want)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func assertBefore(t *testing.T, order []string, before string, after string) {
	t.Helper()

	beforeIndex := -1
	afterIndex := -1

	for i, name := range order {
		switch name {
		case before:
			beforeIndex = i
		case after:
			afterIndex = i
		}
	}

	if beforeIndex == -1 {
		t.Fatalf("expected %q to appear in order %v", before, order)
	}

	if afterIndex == -1 {
		t.Fatalf("expected %q to appear in order %v", after, order)
	}

	if beforeIndex >= afterIndex {
		t.Fatalf("expected %q to appear before %q in order %v", before, after, order)
	}
}

func assertContainsAll(t *testing.T, order []string, expected []string) {
	t.Helper()

	seen := make(map[string]int, len(order))
	for _, name := range order {
		seen[name]++
	}

	for _, name := range expected {
		if seen[name] == 0 {
			t.Fatalf("expected %q to appear in order %v", name, order)
		}
	}

	for name, count := range seen {
		if count > 1 {
			t.Fatalf("expected %q to appear only once in order %v", name, order)
		}
	}
}
