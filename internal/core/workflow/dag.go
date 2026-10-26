package workflow

import (
	"errors"
	"fmt"
)

// ValidateDAG verifies that a workflow pattern defines a valid, cycle-free directed acyclic graph.
func ValidateDAG(pattern Pattern) error {
	if len(pattern.Stages) == 0 {
		return errors.New("workflow pattern contains no stages")
	}
	stageIDs, err := collectStageIDs(pattern.Stages)
	if err != nil {
		return err
	}
	if err := validateDependencies(pattern.Stages, stageIDs); err != nil {
		return err
	}
	return detectCycles(pattern.Stages)
}

func collectStageIDs(stages []Stage) (map[string]bool, error) {
	stageIDs := make(map[string]bool, len(stages))
	for _, s := range stages {
		if s.ID == "" {
			return nil, errors.New("stage ID cannot be empty")
		}
		if s.Role == "" {
			return nil, fmt.Errorf("stage %q has no role specified", s.ID)
		}
		if stageIDs[s.ID] {
			return nil, fmt.Errorf("duplicate stage ID %q", s.ID)
		}
		stageIDs[s.ID] = true
	}
	return stageIDs, nil
}

func validateDependencies(stages []Stage, stageIDs map[string]bool) error {
	for _, s := range stages {
		for _, dep := range s.Needs {
			if dep == s.ID {
				return fmt.Errorf("stage %q cannot depend on itself", s.ID)
			}
			if !stageIDs[dep] {
				return fmt.Errorf("stage %q references unknown dependency %q", s.ID, dep)
			}
		}
	}
	return nil
}

func detectCycles(stages []Stage) error {
	graph := make(map[string][]string, len(stages))
	for _, s := range stages {
		graph[s.ID] = s.Needs
	}
	state := make(map[string]int, len(stages)) // 0: unvisited, 1: visiting, 2: visited
	for _, s := range stages {
		if state[s.ID] == 0 {
			if err := dfsCycle(s.ID, graph, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func dfsCycle(curr string, graph map[string][]string, state map[string]int) error {
	state[curr] = 1
	for _, dep := range graph[curr] {
		if state[dep] == 1 {
			return fmt.Errorf("cycle detected in workflow DAG involving stage %q and %q", curr, dep)
		}
		if state[dep] == 0 {
			if err := dfsCycle(dep, graph, state); err != nil {
				return err
			}
		}
	}
	state[curr] = 2
	return nil
}

// TopologicalSort returns an execution order where all dependencies execute before their dependents.
func TopologicalSort(pattern Pattern) ([]string, error) {
	if err := ValidateDAG(pattern); err != nil {
		return nil, err
	}
	inDegree := make(map[string]int, len(pattern.Stages))
	dependents := make(map[string][]string, len(pattern.Stages))
	for _, s := range pattern.Stages {
		inDegree[s.ID] = len(s.Needs)
		for _, dep := range s.Needs {
			dependents[dep] = append(dependents[dep], s.ID)
		}
	}

	queue := make([]string, 0, len(pattern.Stages))
	for _, s := range pattern.Stages {
		if inDegree[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	order := make([]string, 0, len(pattern.Stages))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		for _, dep := range dependents[curr] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}
	return order, nil
}
