package scheduler

import (
	"container/heap"
	"errors"
)

type scheduler struct {
	deps map[string]map[string]struct{} // task -> dependencies
}

// NewAIScheduler creates a new Scheduler.
func NewAIScheduler() Scheduler {
	return &scheduler{
		deps: make(map[string]map[string]struct{}),
	}
}

func (s *scheduler) AddTask(name string, dependencies ...string) error {
	if name == "" {
		return errors.New("task name must not be empty")
	}

	nextDeps := make(map[string]struct{}, len(dependencies))

	for _, dep := range dependencies {
		if dep == "" {
			return errors.New("dependency name must not be empty")
		}

		if dep == name {
			return errors.New("task may not depend on itself")
		}

		if _, ok := s.deps[dep]; !ok {
			return errors.New("dependency has not been added")
		}

		nextDeps[dep] = struct{}{}
	}

	s.deps[name] = nextDeps
	return nil
}

func (s *scheduler) RemoveTask(name string) {
	delete(s.deps, name)
}

func (s *scheduler) Order() ([]string, error) {
	if len(s.deps) == 0 {
		return []string{}, nil
	}

	indegree := make(map[string]int, len(s.deps))
	reverse := make(map[string][]string, len(s.deps))

	for name := range s.deps {
		indegree[name] = 0
	}

	for name, deps := range s.deps {
		for dep := range deps {
			if _, ok := s.deps[dep]; !ok {
				return nil, errors.New("missing dependency")
			}

			indegree[name]++
			reverse[dep] = append(reverse[dep], name)
		}
	}

	ready := &stringHeap{}
	heap.Init(ready)

	for name, degree := range indegree {
		if degree == 0 {
			heap.Push(ready, name)
		}
	}

	order := make([]string, 0, len(s.deps))

	for ready.Len() > 0 {
		name := heap.Pop(ready).(string)
		order = append(order, name)

		for _, dependent := range reverse[name] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				heap.Push(ready, dependent)
			}
		}
	}

	if len(order) != len(s.deps) {
		return nil, errors.New("cycle detected")
	}

	return order, nil
}

type stringHeap []string

func (h stringHeap) Len() int           { return len(h) }
func (h stringHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h stringHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *stringHeap) Push(x any) {
	*h = append(*h, x.(string))
}

func (h *stringHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
