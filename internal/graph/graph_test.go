package graph

import (
	"strings"
	"testing"

	"github.com/dominionthedev/mushmellow/internal/puff"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

func mustRoot(t *testing.T, puffs map[string]puff.Puff) *workspace.Root {
	t.Helper()
	r := &workspace.Root{Puffs: puffs}
	return r
}

func TestBuild_SimpleChain(t *testing.T) {
	root := mustRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{{Kind: puff.StepShell, Command: "echo a"}}},
		"b": {
			Steps:     []puff.Step{{Kind: puff.StepShell, Command: "echo b"}},
			DependsOn: []puff.DependsOn{{Puff: "a"}},
		},
	})
	// finalize sets Name from map key, which normally happens in
	// ParseFile; do it manually since we're bypassing YAML here.
	for name, p := range root.Puffs {
		p.Name = name
		root.Puffs[name] = p
	}

	g, err := Build(root)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("want 2 nodes, got %d", len(g.Nodes))
	}
	b := g.Nodes["b"]
	if len(b.Edges) != 1 || b.Edges[0].Puff != "a" {
		t.Fatalf("b should depend on a, got %+v", b.Edges)
	}
	if b.Edges[0].EffectiveCondition() != puff.OnSuccess {
		t.Fatalf("default edge condition should be success, got %q", b.Edges[0].EffectiveCondition())
	}
}

func TestBuild_UnknownDependency(t *testing.T) {
	root := mustRoot(t, map[string]puff.Puff{
		"a": {
			Steps:     []puff.Step{{Kind: puff.StepShell, Command: "echo a"}},
			DependsOn: []puff.DependsOn{{Puff: "ghost"}},
		},
	})
	for name, p := range root.Puffs {
		p.Name = name
		root.Puffs[name] = p
	}

	_, err := Build(root)
	if err == nil {
		t.Fatal("expected error for unknown dependency, got nil")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("error should mention the unknown puff name, got: %v", err)
	}
}

func TestBuild_DetectsCycle(t *testing.T) {
	root := mustRoot(t, map[string]puff.Puff{
		"a": {
			Steps:     []puff.Step{{Kind: puff.StepShell, Command: "echo a"}},
			DependsOn: []puff.DependsOn{{Puff: "b"}},
		},
		"b": {
			Steps:     []puff.Step{{Kind: puff.StepShell, Command: "echo b"}},
			DependsOn: []puff.DependsOn{{Puff: "a"}},
		},
	})
	for name, p := range root.Puffs {
		p.Name = name
		root.Puffs[name] = p
	}

	_, err := Build(root)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error should mention cycle, got: %v", err)
	}
}

func TestBuild_BranchExpansionCascadesDownstream(t *testing.T) {
	root := mustRoot(t, map[string]puff.Puff{
		"build": {
			Steps: []puff.Step{{Kind: puff.StepShell, Command: "cargo build"}},
			Branch: &puff.BranchSpec{
				Overrides: map[string][]string{"ENV": {"a", "b"}},
			},
		},
		"test": {
			Steps:     []puff.Step{{Kind: puff.StepShell, Command: "cargo test"}},
			DependsOn: []puff.DependsOn{{Puff: "build"}},
		},
		"unrelated": {
			Steps: []puff.Step{{Kind: puff.StepShell, Command: "echo unrelated"}},
		},
	})
	for name, p := range root.Puffs {
		p.Name = name
		root.Puffs[name] = p
	}

	g, err := Build(root)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// build and test should each be expanded into 2 branch variants;
	// unrelated should be untouched (edge-local scope).
	wantNodes := []string{"build@a", "build@b", "test@a", "test@b", "unrelated"}
	if len(g.Nodes) != len(wantNodes) {
		t.Fatalf("want %d nodes, got %d: %v", len(wantNodes), len(g.Nodes), nodeNames(g))
	}
	for _, name := range wantNodes {
		if _, ok := g.Nodes[name]; !ok {
			t.Errorf("expected node %q, not found. have: %v", name, nodeNames(g))
		}
	}

	// the base template names must not survive expansion.
	if _, ok := g.Nodes["build"]; ok {
		t.Error("template node \"build\" should have been removed after expansion")
	}
	if _, ok := g.Nodes["test"]; ok {
		t.Error("template node \"test\" should have been removed after expansion")
	}

	// test@a must depend on build@a, not the template or the other branch.
	testA := g.Nodes["test@a"]
	if len(testA.Edges) != 1 || testA.Edges[0].Puff != "build@a" {
		t.Fatalf("test@a should depend on build@a, got %+v", testA.Edges)
	}
}

func nodeNames(g *Graph) []string {
	names := make([]string, 0, len(g.Nodes))
	for n := range g.Nodes {
		names = append(names, n)
	}
	return names
}
