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
		panic("invalid target: " + *target)
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

func TestOrderComplexGraphIsDeterministic(t *testing.T) {
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

	assertOrder(t, s, []string{"compile", "lint", "test"})
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
