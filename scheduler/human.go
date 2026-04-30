package scheduler

import (
	"fmt"
	"slices"
	"strings"
)

type task struct {
	name         string
	dependencies []*task
	useCount     int // useCount is how many other tasks have this as a dependency.
	deleted      bool
}

type HumanScheduler struct {
	tasks map[string]*task // tasks maps name to task
}

func NewHumanScheduler() *HumanScheduler {
	return &HumanScheduler{
		tasks: make(map[string]*task),
	}
}

func (s *HumanScheduler) AddTask(name string, dependencies ...string) error {
	if name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	newTask := &task{
		name:         name,
		dependencies: make([]*task, 0, len(dependencies)),
	}

	// Lookup dependencies and add links to them.
	for _, depName := range dependencies {
		if depName == name {
			return fmt.Errorf("task %s cannot have itself as a dependency", name)
		}
		if dep, ok := s.tasks[depName]; !ok {
			return fmt.Errorf("dependency %s not found", depName)
		} else {
			newTask.dependencies = append(newTask.dependencies, dep)
		}
	}

	if oldTask, ok := s.tasks[name]; ok {
		// If it already exists, remove old dependencies.
		for _, dep := range oldTask.dependencies {
			dep.useCount--
		}
		oldTask.dependencies = newTask.dependencies
	} else {
		s.tasks[name] = newTask
	}
	for _, dep := range newTask.dependencies {
		dep.useCount++
	}

	return nil
}

func (s *HumanScheduler) RemoveTask(name string) {
	oldTask, ok := s.tasks[name]
	if !ok {
		return
	}

	if oldTask.useCount == 0 {
		delete(s.tasks, name)
	} else {
		oldTask.deleted = true
	}

	for _, dep := range oldTask.dependencies {
		dep.useCount--
	}
}

func (s *HumanScheduler) Order() ([]string, error) {
	// Look for tasks that were deleted but still needed.
	for _, t := range s.tasks {
		if t.deleted && t.useCount > 0 {
			return nil, fmt.Errorf("dependency %s is missing", t.name)
		}
	}

	// This is based on the algorithm described by Arthur B. Kahn.

	// Instead of prepending processed nodes, we'll append them,
	// then we'll reverse the results at the end.
	reversedResults := make([]string, 0, len(s.tasks))

	// We want to remove root nodes after they are added to the result.
	// This breaks immutability; instead, we log which nodes would be altered.
	// Consider the node removed when t.useCount-visitcount becomes -1.
	visitCounts := make(map[string]int, len(s.tasks))

	for {
		// Collect the root tasks, which are used by exactly 0 other tasks.
		rootTasks := make([]*task, 0, len(s.tasks))
		for _, t := range s.tasks {
			if t.useCount-visitCounts[t.name] == 0 {
				rootTasks = append(rootTasks, t)
			}
		}

		// If there are no root tasks, then we are finished processing.
		if len(rootTasks) == 0 {
			break
		}

		// Virtually remove the root tasks
		for _, t := range rootTasks {
			visitCounts[t.name]++
			for _, dep := range t.dependencies {
				visitCounts[dep.name]++
			}
		}

		// Sort then add the results.
		slices.SortFunc(rootTasks, func(a, b *task) int {
			return strings.Compare(b.name, a.name)
		})
		for _, t := range rootTasks {
			reversedResults = append(reversedResults, t.name)
		}
	}

	// If there are any tasks left, then they are cyclic dependencies.
	for _, t := range s.tasks {
		if t.useCount-visitCounts[t.name] > 0 {
			return nil, fmt.Errorf("task %s has a cyclic dependency", t.name)
		}
	}

	slices.Reverse(reversedResults)
	return reversedResults, nil
}
