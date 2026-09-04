// Package graph builds the executable DAG from a parsed
// workspace.Root: base nodes and edges from declared puffs, static
// branch expansion (cartesian matrix -> sibling puffs, cascaded to
// downstream consumers), and cycle detection. No scheduling, no
// execution — that's internal/scheduler.
package graph

import (
	"fmt"

	"github.com/dominionthedev/mushmellow/internal/puff"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

// Node is one executable puff in the graph — after branch expansion,
// so its Puff.Name is already branch-suffixed where applicable
// (e.g. "cargo_build@env-a").
type Node struct {
	Puff puff.Puff

	// Edges are resolved dependency edges: the exact upstream node
	// names this node waits on, with their conditions. Post-branch
	// expansion, these already point at the correct sibling (e.g. a
	// downstream node depending on a branched puff points at
	// "cargo_build@env-a", not the template name).
	Edges []puff.DependsOn
}

// Graph is the fully expanded, validated executable DAG for one
// workspace.Root (root or a single member — graphs never span
// members, per the no-cross-member-edges rule).
type Graph struct {
	Nodes map[string]*Node
}

// Build constructs and validates the graph: base nodes/edges from
// root.Puffs, static branch expansion with downstream cascade, then
// cycle detection on the final expanded graph.
func Build(root *workspace.Root) (*Graph, error) {
	g := &Graph{Nodes: map[string]*Node{}}

	// Phase 1: base nodes/edges, unexpanded. Also validate every
	// depends_on target actually exists before we do anything else.
	for name, p := range root.Puffs {
		g.Nodes[name] = &Node{Puff: p, Edges: append([]puff.DependsOn(nil), p.DependsOn...)}
	}
	for name, n := range g.Nodes {
		for _, d := range n.Edges {
			if _, ok := g.Nodes[d.Puff]; !ok {
				return nil, fmt.Errorf("puff %q depends_on unknown puff %q", name, d.Puff)
			}
		}
	}

	if err := g.expandBranches(); err != nil {
		return nil, err
	}

	if err := g.detectCycles(); err != nil {
		return nil, err
	}

	return g, nil
}

// expandBranches performs static (matrix-declared) branch expansion.
// v0.1 scope: supports any number of independently branched puffs as
// long as their downstream cascades don't overlap (a puff consuming
// two different branched ancestors — true diamond convergence across
// two branch sources — is undefined behavior for now; the edge-local
// rule handles the *mixed branched+unbranched* case fine, just not
// two independent branch sources converging on one node).
func (g *Graph) expandBranches() error {
	var branchedNames []string
	for name, n := range g.Nodes {
		if n.Puff.Branch != nil {
			branchedNames = append(branchedNames, name)
		}
	}

	for _, baseName := range branchedNames {
		baseNode := g.Nodes[baseName]
		combos := expandMatrix(baseNode.Puff.Branch)
		if len(combos) == 0 {
			continue
		}

		downstream := g.downstreamOf(baseName)

		// Create branch variants of the base node itself.
		variantNames := map[string][]string{baseName: {}} // old name -> new variant names
		for _, combo := range combos {
			v := ExpandPuff(baseNode.Puff, combo.name, combo.overrides, false)
			g.Nodes[v.Name] = &Node{Puff: v, Edges: append([]puff.DependsOn(nil), baseNode.Puff.DependsOn...)}
			variantNames[baseName] = append(variantNames[baseName], v.Name)
		}

		// Create branch variants of every downstream node, rewiring
		// edges: an edge to a branched ancestor points at the
		// matching variant; an edge to anything else (mixed
		// dependency) is left pointing at the original, unbranched
		// node.
		for _, downName := range downstream {
			downNode := g.Nodes[downName]
			for _, combo := range combos {
				v := ExpandPuff(downNode.Puff, combo.name, combo.overrides, false)
				var newEdges []puff.DependsOn
				for _, e := range downNode.Edges {
					if e.Puff == baseName {
						newEdges = append(newEdges, puff.DependsOn{Puff: fmt.Sprintf("%s@%s", baseName, combo.name), Condition: e.Condition})
					} else if names, isBranched := variantNames[e.Puff]; isBranched {
						// upstream is itself a branch-expanded node from
						// an earlier iteration in this same cascade
						for _, vn := range names {
							newEdges = append(newEdges, puff.DependsOn{Puff: vn, Condition: e.Condition})
						}
					} else {
						newEdges = append(newEdges, e)
					}
				}
				g.Nodes[v.Name] = &Node{Puff: v, Edges: newEdges}
			}
		}

		// Remove the now-superseded templates: the base branched
		// puff and every downstream node it cascaded to are replaced
		// entirely by their variants.
		delete(g.Nodes, baseName)
		for _, downName := range downstream {
			delete(g.Nodes, downName)
		}
	}

	return nil
}

// downstreamOf returns every node name transitively reachable by
// following edges forward from name (i.e. every node that depends,
// directly or indirectly, on name).
func (g *Graph) downstreamOf(name string) []string {
	// Build a forward adjacency (upstream -> [downstream consumers])
	// since Edges are stored as "this node depends on that node".
	consumers := map[string][]string{}
	for nodeName, n := range g.Nodes {
		for _, e := range n.Edges {
			consumers[e.Puff] = append(consumers[e.Puff], nodeName)
		}
	}

	seen := map[string]bool{}
	var result []string
	queue := append([]string(nil), consumers[name]...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		result = append(result, cur)
		queue = append(queue, consumers[cur]...)
	}
	return result
}

// detectCycles runs a standard three-color DFS over the dependency
// edges and fails on any back-edge.
func (g *Graph) detectCycles() error {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(g.Nodes))

	var visit func(name string, path []string) error
	visit = func(name string, path []string) error {
		switch color[name] {
		case black:
			return nil
		case gray:
			return fmt.Errorf("dependency cycle detected: %s -> %s", joinPath(path), name)
		}
		color[name] = gray
		node, ok := g.Nodes[name]
		if !ok {
			return fmt.Errorf("edge references unknown node %q", name)
		}
		for _, e := range node.Edges {
			if err := visit(e.Puff, append(path, name)); err != nil {
				return err
			}
		}
		color[name] = black
		return nil
	}

	for name := range g.Nodes {
		if color[name] == white {
			if err := visit(name, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func joinPath(path []string) string {
	out := ""
	for i, p := range path {
		if i > 0 {
			out += " -> "
		}
		out += p
	}
	return out
}
