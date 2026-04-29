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

// Resolve returns the puffs in topological order, grouped into execution batches
func (r *Resolver) Resolve() ([][]config.Puff, error) {
	if r.graph.HasCycle() {
		return nil, ErrCycleDetected
	}

	// Calculate in-degrees
	inDegree := make(map[string]int)
	for id, node := range r.graph.Nodes {
		inDegree[id] = len(node.Edges)
		for _, dep := range node.Edges {
			if _, ok := r.graph.Nodes[dep]; !ok {
				return nil, fmt.Errorf("%w: puff '%s' depends on non-existent puff '%s'", ErrMissingDependency, id, dep)
			}
		}
	}

	var batches [][]config.Puff

	for {
		// Find all nodes with in-degree 0
		var currentBatch []config.Puff
		var currentIDs []string

		for id, degree := range inDegree {
			if degree == 0 {
				currentBatch = append(currentBatch, r.graph.Nodes[id].Puff)
				currentIDs = append(currentIDs, id)
			}
		}

		if len(currentBatch) == 0 {
			break
		}

		batches = append(batches, currentBatch)

		// Remove current batch from in-degree map and update neighbors
		for _, id := range currentIDs {
			delete(inDegree, id)
			for _, dependentID := range r.graph.Reverse[id] {
				inDegree[dependentID]--
			}
		}
	}

	if len(inDegree) > 0 {
		return nil, fmt.Errorf("could not resolve all dependencies (hidden cycle)")
	}

	return batches, nil
}
