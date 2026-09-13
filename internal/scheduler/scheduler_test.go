package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dominionthedev/mushmellow/internal/graph"
	"github.com/dominionthedev/mushmellow/internal/puff"
	"github.com/dominionthedev/mushmellow/internal/state"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

func newTestRoot(t *testing.T, puffs map[string]puff.Puff) *workspace.Root {
	t.Helper()
	for name, p := range puffs {
		p.Name = name
		puffs[name] = p
	}
	return &workspace.Root{
		Dir:   t.TempDir(),
		Puffs: puffs,
	}
}

func buildAndRun(t *testing.T, root *workspace.Root, invoke string) *state.Run {
	t.Helper()
	g, err := graph.Build(root)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	result, err := New(root, g).Run(invoke)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	return result
}

func shell(cmd string) puff.Step { return puff.Step{Kind: puff.StepShell, Command: cmd} }

func TestRun_SimpleChain_Success(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
		"b": {Steps: []puff.Step{shell("true")}, DependsOn: []puff.DependsOn{{Puff: "a"}}},
	})
	r := buildAndRun(t, root, "b")
	if r.Puffs["a"].Status != state.Success || r.Puffs["b"].Status != state.Success {
		t.Fatalf("want both success, got a=%s b=%s", r.Puffs["a"].Status, r.Puffs["b"].Status)
	}
}

func TestRun_FailurePropagation_BlocksDownstream(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
		"b": {Steps: []puff.Step{shell("false")}, DependsOn: []puff.DependsOn{{Puff: "a"}}},
		"c": {Steps: []puff.Step{shell("true")}, DependsOn: []puff.DependsOn{{Puff: "b"}}},
	})
	r := buildAndRun(t, root, "c")

	if r.Puffs["a"].Status != state.Success {
		t.Errorf("a: want success, got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["b"].Status != state.Failed {
		t.Errorf("b: want failed, got %s", r.Puffs["b"].Status)
	}
	if r.Puffs["c"].Status != state.Blocked {
		t.Errorf("c: want blocked, got %s", r.Puffs["c"].Status)
	}
	if r.Puffs["c"].BlockedBy != "b" {
		t.Errorf("c.BlockedBy: want %q, got %q", "b", r.Puffs["c"].BlockedBy)
	}
}

func TestRun_ExternalOnFailureDependent_Fires(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("false")}},
		"notify": {
			Steps:     []puff.Step{shell("true")},
			DependsOn: []puff.DependsOn{{Puff: "a", Condition: puff.OnFailureCond}},
		},
	})
	r := buildAndRun(t, root, "notify")

	if r.Puffs["a"].Status != state.Failed {
		t.Errorf("a: want failed, got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["notify"].Status != state.Success {
		t.Errorf("notify: want success (on:failure edge should fire), got %s", r.Puffs["notify"].Status)
	}
}

func TestRun_OnFailureDependent_NeverFiresIfUpstreamSucceeds(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
		"notify": {
			Steps:     []puff.Step{shell("true")},
			DependsOn: []puff.DependsOn{{Puff: "a", Condition: puff.OnFailureCond}},
		},
	})
	r := buildAndRun(t, root, "notify")

	// a succeeded, so notify's on:failure condition can never be
	// satisfied - it should end up Blocked (the "doomed" case), not
	// silently stuck Pending forever.
	if r.Puffs["notify"].Status != state.Blocked {
		t.Errorf("notify: want blocked (condition unsatisfiable), got %s", r.Puffs["notify"].Status)
	}
}

func TestRun_SelfHandler_NoSwallowByDefault(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps:     []puff.Step{shell("false")},
			OnFailure: &puff.OnFailure{Steps: []puff.Step{shell("true")}}, // Recover defaults false
		},
	})
	r := buildAndRun(t, root, "a")

	if r.Puffs["a"].Status != state.Failed {
		t.Fatalf("want failed (handler must not swallow by default), got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["a"].SelfHandler == nil || !r.Puffs["a"].SelfHandler.Fired {
		t.Fatal("want self_handler.fired=true")
	}
	if r.Puffs["a"].SelfHandler.Recovered {
		t.Fatal("want self_handler.recovered=false")
	}
}

func TestRun_SelfHandler_ExplicitRecover_UnblocksDownstream(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps:     []puff.Step{shell("false")},
			OnFailure: &puff.OnFailure{Steps: []puff.Step{shell("true")}, Recover: true},
		},
		"b": {Steps: []puff.Step{shell("true")}, DependsOn: []puff.DependsOn{{Puff: "a"}}},
	})
	r := buildAndRun(t, root, "b")

	if r.Puffs["a"].Status != state.Recovered {
		t.Fatalf("a: want recovered, got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["b"].Status != state.Success {
		t.Fatalf("b: want success (recovered satisfies on:success), got %s", r.Puffs["b"].Status)
	}
}

func TestRun_Retries_Exhausted(t *testing.T) {
	orig := backoffFunc
	backoffFunc = func(int) time.Duration { return time.Millisecond }
	defer func() { backoffFunc = orig }()

	two := 2
	root := newTestRoot(t, map[string]puff.Puff{
		"flaky": {Steps: []puff.Step{shell("false")}, Retries: &two},
	})
	r := buildAndRun(t, root, "flaky")

	if r.Puffs["flaky"].Status != state.Failed {
		t.Fatalf("want failed, got %s", r.Puffs["flaky"].Status)
	}
	if len(r.Puffs["flaky"].Attempts) != 3 { // 1 initial + 2 retries
		t.Fatalf("want 3 attempts, got %d", len(r.Puffs["flaky"].Attempts))
	}
}

func TestRun_UnrelatedPuffOutsideClosure_NotIncluded(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a":         {Steps: []puff.Step{shell("true")}},
		"unrelated": {Steps: []puff.Step{shell("true")}},
	})
	r := buildAndRun(t, root, "a")

	if _, ok := r.Puffs["unrelated"]; ok {
		t.Fatal("unrelated puff should not be part of a's run at all - closure is upstream-only")
	}
}

func TestRun_StateFilePersisted(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
	})
	r := buildAndRun(t, root, "a")

	path := filepath.Join(root.Dir, ".mushmellow", "runs", r.RunID, "state.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected state file at %s: %v", path, err)
	}
}

func writeMemberWorkspace(t *testing.T, rootDir, memberRelDir string, yaml string) {
	t.Helper()
	dir := filepath.Join(rootDir, memberRelDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir member dir: %v", err)
	}
	name := filepath.Base(memberRelDir) + ".mushmellow.yaml"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write member yaml: %v", err)
	}
}

func TestRun_MemberCall_Success(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"build": {Steps: []puff.Step{{Kind: puff.StepMember, Member: "cmd/api", MemberPuff: "build"}}},
	})
	writeMemberWorkspace(t, root.Dir, "cmd/api", "puffs:\n  build:\n    steps:\n      - run: \"true\"\n")

	r := buildAndRun(t, root, "build")

	if r.Puffs["build"].Status != state.Success {
		t.Fatalf("want success, got %s", r.Puffs["build"].Status)
	}
	calls := r.Puffs["build"].MemberCalls
	if len(calls) != 1 || calls[0].Member != "cmd/api" || calls[0].Puff != "build" || calls[0].Status != state.Success {
		t.Fatalf("want one successful member call record, got %+v", calls)
	}
	// member state must be its own separate file, not inlined.
	memberStatePath := filepath.Join(root.Dir, "cmd/api", ".mushmellow", "runs", calls[0].RunID, "state.json")
	if _, err := os.Stat(memberStatePath); err != nil {
		t.Fatalf("expected member's own state file at %s: %v", memberStatePath, err)
	}
}

func TestRun_MemberCall_FailurePropagatesToParent(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"build": {Steps: []puff.Step{{Kind: puff.StepMember, Member: "cmd/api", MemberPuff: "build"}}},
	})
	writeMemberWorkspace(t, root.Dir, "cmd/api", "puffs:\n  build:\n    steps:\n      - run: \"false\"\n")

	r := buildAndRun(t, root, "build")

	if r.Puffs["build"].Status != state.Failed {
		t.Fatalf("want parent puff to fail when member puff fails, got %s", r.Puffs["build"].Status)
	}
	if len(r.Puffs["build"].MemberCalls) != 1 || r.Puffs["build"].MemberCalls[0].Status != state.Failed {
		t.Fatalf("want member call recorded as failed, got %+v", r.Puffs["build"].MemberCalls)
	}
}
