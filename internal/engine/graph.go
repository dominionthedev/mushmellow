package engine

import (
	"errors"

	"github.com/dominionthedev/mushmellow/internal/config"
)

var (
	ErrCycleDetected       = errors.New("dependency cycle detected")
	ErrMissingDependency   = errors.New("missing dependency")
)

// Graph represents the dependency graph (DAG) for puffs
type Graph struct {
	Nodes   map[string]*Node
	Reverse map[string][]string // puffID -> puffs that depend on it
}

// Node represents a puff in the DAG
type Node struct {
	ID    string
	Puff  config.Puff
	Edges []string // IDs of puffs this node depends on
}

// NewGraph creates a new dependency graph from a mushmellow
func NewGraph(mushmellow config.Mushmellow) *Graph {
	g := &Graph{
		Nodes:   make(map[string]*Node),
		Reverse: make(map[string][]string),
	}

	for _, puff := range mushmellow.Puffs {
		g.Nodes[puff.ID] = &Node{
			ID:    puff.ID,
			Puff:  puff,
			Edges: puff.DependsOn,
		}

		for _, dep := range puff.DependsOn {
			g.Reverse[dep] = append(g.Reverse[dep], puff.ID)
		}
	}

	return g
}

// HasCycle checks if the graph contains a cycle using DFS
func (g *Graph) HasCycle() bool {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var dfs func(string) bool
	dfs = func(id string) bool {
		visited[id] = true
		recursionStack[id] = true

		node, ok := g.Nodes[id]
		if !ok {
			// Dependency on a non-existent node is handled by resolver, 
			// but for cycle detection we just skip.
			delete(recursionStack, id)
			return false
		}

		for _, dep := range node.Edges {
			if recursionStack[dep] {
				return true
			}
			if !visited[dep] && dfs(dep) {
				return true
			}
		}

		delete(recursionStack, id)
		return false
	}

	for id := range g.Nodes {
		if !visited[id] {
			if dfs(id) {
				return true
			}
		}
	}

	return false
}
