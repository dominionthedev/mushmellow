package engine

import (
	"fmt"

	"github.com/dominionthedev/mushmellow/internal/config"
)

// Resolver resolves the execution order of puffs
type Resolver struct {
	graph *Graph
}

// NewResolver creates a new Resolver
func NewResolver(mushmellow config.Mushmellow) *Resolver {
	return &Resolver{
		graph: NewGraph(mushmellow),
	}
}

// Resolve returns the puffs in topological order
func (r *Resolver) Resolve() ([]config.Puff, error) {
	if r.graph.HasCycle() {
		return nil, ErrCycleDetected
	}

	// Calculate in-degrees (number of dependencies for each node)
	inDegree := make(map[string]int)
	for id, node := range r.graph.Nodes {
		inDegree[id] = len(node.Edges)
		
		// Validate that all dependencies exist
		for _, dep := range node.Edges {
			if _, ok := r.graph.Nodes[dep]; !ok {
				return nil, fmt.Errorf("%w: puff '%s' depends on non-existent puff '%s'", ErrMissingDependency, id, dep)
			}
		}
	}

	// Initialize queue with nodes that have no dependencies
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	resultIDs := make([]string, 0)
	for len(queue) > 0 {
		// We want deterministic sorting if multiple nodes are ready.
		// For now, we just take the first. 
		// In the future, we might sort by ID or priority.
		currentID := queue[0]
		queue = queue[1:]
		resultIDs = append(resultIDs, currentID)

		// Find nodes that depend on the current node
		for _, dependentID := range r.graph.Reverse[currentID] {
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				queue = append(queue, dependentID)
			}
		}
	}

	if len(resultIDs) != len(r.graph.Nodes) {
		// This should have been caught by HasCycle, but good to have as a fallback.
		return nil, fmt.Errorf("could not resolve all dependencies (possible hidden cycle or logic error)")
	}

	// Map IDs back to Puff objects
	puffs := make([]config.Puff, 0, len(resultIDs))
	for _, id := range resultIDs {
		puffs = append(puffs, r.graph.Nodes[id].Puff)
	}

	return puffs, nil
}
