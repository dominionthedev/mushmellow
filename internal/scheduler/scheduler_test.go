package scheduler

import (
	"net"
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
	if r.Puffs["c"].Reason != "b" {
		t.Errorf("c.Reason: want %q, got %q", "b", r.Puffs["c"].Reason)
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

func TestRun_OnFailureDependent_UpstreamSucceeds_EndsSkipped(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
		"notify": {
			Steps:     []puff.Step{shell("true")},
			DependsOn: []puff.DependsOn{{Puff: "a", Condition: puff.OnFailureCond}},
		},
	})
	r := buildAndRun(t, root, "notify")

	// a succeeded - nothing failed, so notify's on:failure condition
	// can never be satisfied. This must be Skipped, not Blocked:
	// nothing broke, the watched-for failure simply never happened.
	if r.Puffs["notify"].Status != state.Skipped {
		t.Errorf("notify: want skipped (condition unsatisfiable, but nothing failed), got %s", r.Puffs["notify"].Status)
	}
}

func TestRun_OnFailureDependent_FiresOnRecovered(t *testing.T) {
	// A puff that failed and was then explicitly recovered by its own
	// self-handler still failed first. An external on:"failure"
	// watcher exists to react to failures even ones handled
	// internally too - it should still fire.
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps:     []puff.Step{shell("false")},
			OnFailure: &puff.OnFailure{Steps: []puff.Step{shell("true")}, Recover: true},
		},
		"notify": {
			Steps:     []puff.Step{shell("true")},
			DependsOn: []puff.DependsOn{{Puff: "a", Condition: puff.OnFailureCond}},
		},
	})
	r := buildAndRun(t, root, "notify")

	if r.Puffs["a"].Status != state.Recovered {
		t.Fatalf("a: want recovered, got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["notify"].Status != state.Success {
		t.Errorf("notify: want success (on:failure should fire even though a recovered), got %s", r.Puffs["notify"].Status)
	}
}

func TestRun_OnSuccessDependent_UpstreamSkipped_CascadesSkipped(t *testing.T) {
	// b's on:failure condition is unsatisfiable (a succeeds) -> b ends
	// Skipped. c depends on b for on:"success" -> c must inherit
	// Skipped too, not be mislabeled Blocked (nothing failed anywhere
	// in this chain).
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("true")}},
		"b": {
			Steps:     []puff.Step{shell("true")},
			DependsOn: []puff.DependsOn{{Puff: "a", Condition: puff.OnFailureCond}},
		},
		"c": {Steps: []puff.Step{shell("true")}, DependsOn: []puff.DependsOn{{Puff: "b"}}},
	})
	r := buildAndRun(t, root, "c")

	if r.Puffs["b"].Status != state.Skipped {
		t.Fatalf("b: want skipped, got %s", r.Puffs["b"].Status)
	}
	if r.Puffs["c"].Status != state.Skipped {
		t.Errorf("c: want skipped (cascaded from b, nothing failed), got %s", r.Puffs["c"].Status)
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

func TestRun_When_FileContains_Met(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When: []puff.Criterion{{
				Kind: puff.CriterionFileContains, Path: "marker.txt", Substr: "// TEST:",
			}},
		},
	})
	if err := os.WriteFile(filepath.Join(root.Dir, "marker.txt"), []byte("// TEST: ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := buildAndRun(t, root, "a")
	if r.Puffs["a"].Status != state.Success {
		t.Fatalf("want success, got %s", r.Puffs["a"].Status)
	}
}

func TestRun_When_FileContains_Unmet_EndsSkipped(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When: []puff.Criterion{{
				Kind: puff.CriterionFileContains, Path: "marker.txt", Substr: "// TEST:",
			}},
		},
	})
	if err := os.WriteFile(filepath.Join(root.Dir, "marker.txt"), []byte("nothing here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := buildAndRun(t, root, "a")
	if r.Puffs["a"].Status != state.Skipped {
		t.Fatalf("want skipped (criterion unmet, nothing failed), got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["a"].Reason == "" {
		t.Fatal("want a human-readable reason")
	}
}

func TestRun_When_FileContains_MissingFile_IsUnmetNotError(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When: []puff.Criterion{{
				Kind: puff.CriterionFileContains, Path: "does-not-exist.txt", Substr: "x",
			}},
		},
	})
	r := buildAndRun(t, root, "a") // must not error out the whole run
	if r.Puffs["a"].Status != state.Skipped {
		t.Fatalf("want skipped, got %s", r.Puffs["a"].Status)
	}
}

func TestRun_When_EnvSet(t *testing.T) {
	t.Setenv("MUSHMELLOW_TEST_VAR", "1")
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When:  []puff.Criterion{{Kind: puff.CriterionEnvSet, EnvVar: "MUSHMELLOW_TEST_VAR"}},
		},
	})
	r := buildAndRun(t, root, "a")
	if r.Puffs["a"].Status != state.Success {
		t.Fatalf("want success, got %s", r.Puffs["a"].Status)
	}
}

func TestRun_When_EnvEquals_WrongValue_EndsSkipped(t *testing.T) {
	t.Setenv("MUSHMELLOW_TEST_VAR", "wrong")
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When: []puff.Criterion{{
				Kind: puff.CriterionEnvEquals, EnvVar: "MUSHMELLOW_TEST_VAR", EnvValue: "expected",
			}},
		},
	})
	r := buildAndRun(t, root, "a")
	if r.Puffs["a"].Status != state.Skipped {
		t.Fatalf("want skipped, got %s", r.Puffs["a"].Status)
	}
}

func TestRun_When_CommandOK(t *testing.T) {
	root := newTestRoot(t, map[string]puff.Puff{
		"true_case":  {Steps: []puff.Step{shell("true")}, When: []puff.Criterion{{Kind: puff.CriterionCommandOK, Command: "true"}}},
		"false_case": {Steps: []puff.Step{shell("true")}, When: []puff.Criterion{{Kind: puff.CriterionCommandOK, Command: "false"}}},
	})
	g, err := graph.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	d := New(root, g)

	r1, err := d.Run("true_case")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Puffs["true_case"].Status != state.Success {
		t.Fatalf("true_case: want success, got %s", r1.Puffs["true_case"].Status)
	}

	d2 := New(root, g)
	r2, err := d2.Run("false_case")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Puffs["false_case"].Status != state.Skipped {
		t.Fatalf("false_case: want skipped, got %s", r2.Puffs["false_case"].Status)
	}
}

func TestRun_When_PortOpen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open a real listener for the test: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	root := newTestRoot(t, map[string]puff.Puff{
		"open_port": {
			Steps: []puff.Step{shell("true")},
			When:  []puff.Criterion{{Kind: puff.CriterionPortOpen, Host: "127.0.0.1", Port: port}},
		},
	})
	r := buildAndRun(t, root, "open_port")
	if r.Puffs["open_port"].Status != state.Success {
		t.Fatalf("want success (port is actually open), got %s", r.Puffs["open_port"].Status)
	}
}

func TestRun_When_PortClosed_EndsSkipped(t *testing.T) {
	// Grab a port and immediately close the listener, so nothing is
	// actually bound - a real "not reachable" case, not a guess.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate a port for the test: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	root := newTestRoot(t, map[string]puff.Puff{
		"closed_port": {
			Steps: []puff.Step{shell("true")},
			When:  []puff.Criterion{{Kind: puff.CriterionPortOpen, Host: "127.0.0.1", Port: port}},
		},
	})
	r := buildAndRun(t, root, "closed_port")
	if r.Puffs["closed_port"].Status != state.Skipped {
		t.Fatalf("want skipped (port not reachable, nothing failed), got %s", r.Puffs["closed_port"].Status)
	}
}

func TestRun_When_NoDependencies_StillChecked(t *testing.T) {
	// Regression: evaluateReadiness used to short-circuit to
	// ready=true for any puff with zero depends_on edges, skipping
	// When entirely. A puff with no dependencies but a when: block
	// must still have it evaluated.
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {
			Steps: []puff.Step{shell("true")},
			When:  []puff.Criterion{{Kind: puff.CriterionEnvSet, EnvVar: "MUSHMELLOW_DEFINITELY_UNSET_VAR"}},
		},
	})
	r := buildAndRun(t, root, "a")
	if r.Puffs["a"].Status != state.Skipped {
		t.Fatalf("want skipped - when: must be checked even with no depends_on, got %s", r.Puffs["a"].Status)
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

func TestRun_Halt_FailedDependencyStillBlocked_NotCancelled(t *testing.T) {
	// b is blocked because its own dependency (a) actually failed -
	// that's ordinary blocking and has nothing to do with halt mode.
	// Halt only changes what happens to *independent* puffs that
	// would otherwise keep running; it doesn't relabel a real
	// dependency failure as a cancellation.
	root := newTestRoot(t, map[string]puff.Puff{
		"a": {Steps: []puff.Step{shell("false")}},
		"b": {Steps: []puff.Step{shell("true")}, DependsOn: []puff.DependsOn{{Puff: "a"}}},
	})
	root.OnFailure = workspace.Halt

	r := buildAndRun(t, root, "b")

	if r.Puffs["a"].Status != state.Failed {
		t.Fatalf("a: want failed, got %s", r.Puffs["a"].Status)
	}
	if r.Puffs["b"].Status != state.Blocked {
		t.Fatalf("b: want blocked (its own dependency failed, unrelated to halt), got %s", r.Puffs["b"].Status)
	}
}

func TestExecute_Halted_CancelsIndependentPuff(t *testing.T) {
	// White-box test of the actual halt mechanism: a puff with no
	// failed dependency of its own, but caught by halt before it got
	// a chance to run, should be marked Cancelled - not run, and not
	// mislabeled as Blocked/Failed. Driven directly at the execute()
	// level since deterministically racing real concurrent dispatch
	// against a halt trigger would be flaky.
	root := newTestRoot(t, map[string]puff.Puff{
		"c": {Steps: []puff.Step{shell("true")}},
	})
	g, err := graph.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	closure, err := g.Closure("c")
	if err != nil {
		t.Fatal(err)
	}

	d := New(root, g)
	d.closure = closure
	d.State = &state.Run{
		RunID: "test",
		Puffs: map[string]*state.PuffState{"c": {Status: state.Pending}},
	}
	d.pools = map[string]chan struct{}{"mushmellow": make(chan struct{}, 1)}
	d.halted = true // simulate halt already in effect before c ever dispatched

	d.execute("c", closure["c"])

	if d.State.Puffs["c"].Status != state.Cancelled {
		t.Fatalf("want cancelled, got %s", d.State.Puffs["c"].Status)
	}
	if d.State.Puffs["c"].Reason == "" {
		t.Fatal("want a reason explaining the cancellation")
	}
}
