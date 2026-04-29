package engine

import (
	"errors"
	"testing"

	"github.com/dominionthedev/mushmellow/internal/config"
)

func TestGraph_HasCycle(t *testing.T) {
	tests := []struct {
		name     string
		puffs    []config.Puff
		wantCycle bool
	}{
		{
			name: "linear dependencies",
			puffs: []config.Puff{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"C"}},
				{ID: "C"},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			puffs: []config.Puff{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"A"}},
			},
			wantCycle: true,
		},
		{
			name: "deep cycle",
			puffs: []config.Puff{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"C"}},
				{ID: "C", DependsOn: []string{"A"}},
			},
			wantCycle: true,
		},
		{
			name: "no dependencies",
			puffs: []config.Puff{
				{ID: "A"},
				{ID: "B"},
			},
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := config.Mushmellow{Puffs: tt.puffs}
			g := NewGraph(m)
			if got := g.HasCycle(); got != tt.wantCycle {
				t.Errorf("Graph.HasCycle() = %v, want %v", got, tt.wantCycle)
			}
		})
	}
}

func TestResolver_Resolve(t *testing.T) {
	tests := []struct {
		name       string
		puffs      []config.Puff
		wantBatches int
		wantErr    error
	}{
		{
			name: "linear dependencies",
			puffs: []config.Puff{
				{ID: "build", DependsOn: []string{"test"}},
				{ID: "test", DependsOn: []string{"lint"}},
				{ID: "lint"},
			},
			wantBatches: 3, // lint, then test, then build
		},
		{
			name: "missing dependency",
			puffs: []config.Puff{
				{ID: "A", DependsOn: []string{"B"}},
			},
			wantErr: ErrMissingDependency,
		},
		{
			name: "parallel groups",
			puffs: []config.Puff{
				{ID: "A"},
				{ID: "B"},
				{ID: "C", DependsOn: []string{"A", "B"}},
			},
			wantBatches: 2, // {A, B}, then {C}
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := config.Mushmellow{Puffs: tt.puffs}
			r := NewResolver(m)
			got, err := r.Resolve()

			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolver.Resolve() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Resolver.Resolve() unexpected error: %v", err)
			}

			if len(got) != tt.wantBatches {
				t.Fatalf("Resolver.Resolve() got %d batches, want %d", len(got), tt.wantBatches)
			}
		})
	}
}
