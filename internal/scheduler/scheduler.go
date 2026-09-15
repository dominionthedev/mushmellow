// Package scheduler executes a graph.Graph: readiness evaluation per
// edge condition, per-profile worker pools, retries, self-handlers,
// member-call delegation, and isolate/halt failure propagation. This
// is where Runtime State (internal/state) gets written.
package scheduler

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/dominionthedev/mushmellow/internal/graph"
	"github.com/dominionthedev/mushmellow/internal/puff"
	"github.com/dominionthedev/mushmellow/internal/state"
	"github.com/dominionthedev/mushmellow/internal/workspace"
)

// haltGrace is how long a SIGTERM'd process gets before SIGKILL on
// halt. Deliberately short — halt means "stop now", not "wind down".
const haltGrace = 3 * time.Second

// Dispatcher executes one invocation's dependency closure.
type Dispatcher struct {
	Root  *workspace.Root
	Graph *graph.Graph

	mu      sync.Mutex // guards State and pools bookkeeping below
	State   *state.Run
	closure map[string]*graph.Node
	pools   map[string]chan struct{}

	runningCmds sync.Map // node name -> *exec.Cmd, for halt cancellation
	halted      bool
}

// New builds a Dispatcher over an already-built graph.
func New(root *workspace.Root, g *graph.Graph) *Dispatcher {
	return &Dispatcher{Root: root, Graph: g}
}

// Run executes invokedPuff's full dependency closure to completion,
// persisting Runtime State as it goes. Returns the final state even
// on failure — a failed run is not a Go error, it's a status.
func (d *Dispatcher) Run(invokedPuff string) (*state.Run, error) {
	closure, err := d.Graph.Closure(invokedPuff)
	if err != nil {
		return nil, err
	}
	d.closure = closure

	d.State = &state.Run{
		RunID:         state.NewRunID(),
		InvokedPuff:   invokedPuff,
		Workspace:     d.Root.Dir,
		OnFailureMode: string(d.Root.EffectiveOnFailure()),
		StartedAt:     time.Now(),
		Puffs:         map[string]*state.PuffState{},
	}

	base := d.Root.BaseProfile()
	d.pools = map[string]chan struct{}{}
	for name, node := range closure {
		prof := d.resolveProfile(node.Puff.Profile)
		pool := puff.ResolvePool(&node.Puff, prof, &base)
		if _, ok := d.pools[prof.Name]; !ok {
			d.pools[prof.Name] = make(chan struct{}, pool)
		}
		d.State.Puffs[name] = &state.PuffState{
			Status:       state.Pending,
			Profile:      prof.Name,
			ResolvedPool: pool,
			Branch:       node.Puff.BranchName,
			MaxRetries:   d.resolveRetries(&node.Puff),
		}
	}

	doneCh := make(chan string, len(closure))
	dispatched := map[string]bool{}

	var dispatchReady func()
	dispatchReady = func() {
		for name, node := range closure {
			ps := d.State.Puffs[name]
			d.mu.Lock()
			terminal := ps.Status.IsTerminal()
			d.mu.Unlock()
			if dispatched[name] || terminal {
				continue
			}
			ready, reason, doomedStatus := d.evaluateReadiness(node)
			if doomedStatus != "" {
				d.mu.Lock()
				ps.Status = doomedStatus
				ps.Reason = reason
				d.mu.Unlock()
				doneCh <- name // trigger a rescan of its own consumers
				continue
			}
			if !ready {
				continue
			}
			dispatched[name] = true
			d.mu.Lock()
			ps.Status = state.Running
			ps.StartedAt = time.Now()
			d.mu.Unlock()
			go func(n string, nd *graph.Node) {
				d.execute(n, nd)
				doneCh <- n
			}(name, node)
		}
	}

	dispatchReady()
	completed := 0
	for completed < len(closure) {
		<-doneCh
		completed = d.countTerminal(closure)
		dispatchReady()
	}

	d.State.EndedAt = time.Now()
	if err := d.State.Save(d.Root.Dir); err != nil {
		return d.State, fmt.Errorf("saving run state: %w", err)
	}
	return d.State, nil
}

func (d *Dispatcher) countTerminal(closure map[string]*graph.Node) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for name := range closure {
		if d.State.Puffs[name].Status.IsTerminal() {
			n++
		}
	}
	return n
}

// evaluateReadiness checks every edge of node against current
// (possibly still-pending) upstream status. Returns ready=true only
// once every edge is both terminal and satisfied, AND every When
// precondition (if any) is met at that moment.
//
// When ready is false and doomedStatus is non-empty, this node will
// never run - dispatch should record doomedStatus (Blocked, Skipped,
// or Cancelled) rather than leaving it Pending forever:
//   - Blocked: a real failure happened - either the direct upstream
//     Failed, or it was itself Blocked (cascading a failure further
//     up the chain).
//   - Skipped: nothing failed. An on:"failure" edge whose upstream
//     never actually failed (it succeeded, or was itself
//     Skipped/Cancelled), or a when: precondition that wasn't met.
//   - Cancelled: the upstream was Cancelled by a halt, and this node
//     inherits that rather than being mislabeled as a failure.
//
// When both ready and doomedStatus are empty/false, at least one edge
// is still pending - try again once more nodes complete.
func (d *Dispatcher) evaluateReadiness(node *graph.Node) (ready bool, reason string, doomedStatus state.Status) {
	d.mu.Lock()
	allTerminal := true
	for _, e := range node.Edges {
		up := d.State.Puffs[e.Puff]
		if !up.Status.IsTerminal() {
			allTerminal = false
			continue
		}
		cond := e.EffectiveCondition()
		if !edgeSatisfied(cond, up.Status) {
			d.mu.Unlock()
			if cond == puff.OnFailureCond {
				// the failure this puff was watching for never
				// happened at that exact upstream node - nothing
				// broke, so this is a skip, not a block.
				return false, fmt.Sprintf("on:\"failure\" never met - %s ended %s", e.Puff, up.Status), state.Skipped
			}
			return false, e.Puff, state.CascadeStatus(up.Status)
		}
	}
	d.mu.Unlock()
	if !allTerminal {
		return false, "", ""
	}

	// Criteria evaluation can block on I/O (file reads, exec,
	// network dials up to a few seconds for port_open) - it must not
	// run while holding d.mu, or every other goroutine trying to
	// update its own PuffState stalls behind it.
	if len(node.Puff.When) > 0 {
		met, unmetReason := evaluateCriteria(d.Root.Dir, node.Puff.When)
		if !met {
			return false, unmetReason, state.Skipped
		}
	}
	return true, "", ""
}

// evaluateCriteria checks every When precondition on node, in order,
// stopping at the first unmet one. Evaluated exactly once, at the
// moment the node would otherwise dispatch (all edges already
// satisfied) - not polled repeatedly and not pre-checked at parse
// time, since the world (a file's contents, an env var, a port) can
// change between parse and dispatch. A snapshot taken once here is
// the answer committed to; this node will not be re-checked later if
// the world changes after this call.
func evaluateCriteria(workspaceDir string, criteria []puff.Criterion) (met bool, unmetReason string) {
	for _, c := range criteria {
		ok, err := evaluateCriterion(workspaceDir, c)
		if err != nil {
			return false, fmt.Sprintf("when: %s (error: %v)", c.Describe(), err)
		}
		if !ok {
			return false, fmt.Sprintf("when: %s", c.Describe())
		}
	}
	return true, ""
}

func evaluateCriterion(workspaceDir string, c puff.Criterion) (bool, error) {
	switch c.Kind {
	case puff.CriterionFileContains:
		path := c.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspaceDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		return bytes.Contains(data, []byte(c.Substr)), nil

	case puff.CriterionEnvSet:
		_, ok := os.LookupEnv(c.EnvVar)
		return ok, nil

	case puff.CriterionEnvEquals:
		v, ok := os.LookupEnv(c.EnvVar)
		return ok && v == c.EnvValue, nil

	case puff.CriterionCommandOK:
		cmd := exec.Command("sh", "-c", c.Command)
		return cmd.Run() == nil, nil

	case puff.CriterionPortOpen:
		addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return false, nil // unreachable is "not met", not an error
		}
		_ = conn.Close()
		return true, nil

	default:
		return false, fmt.Errorf("unknown criterion kind %q", c.Kind)
	}
}

func edgeSatisfied(cond puff.EdgeCondition, upstream state.Status) bool {
	switch cond {
	case puff.OnAlways:
		return true
	case puff.OnFailureCond:
		return upstream.SatisfiesFailure()
	default: // OnSuccess
		return upstream.SatisfiesSuccess()
	}
}

// isExempt reports whether node should be allowed to run/finish
// during a halt: puffs whose readiness depends on failure or always
// conditions are the cleanup/notify puffs the halt design explicitly
// carves out.
func (d *Dispatcher) isExempt(node *graph.Node) bool {
	for _, e := range node.Edges {
		if e.EffectiveCondition() == puff.OnFailureCond || e.EffectiveCondition() == puff.OnAlways {
			return true
		}
	}
	return false
}

func (d *Dispatcher) resolveProfile(name string) *puff.Profile {
	if name == "" {
		p := d.Root.BaseProfile()
		return &p
	}
	if p, ok := d.Root.Profiles[name]; ok {
		return &p
	}
	p := d.Root.BaseProfile()
	return &p
}

func (d *Dispatcher) resolveRetries(p *puff.Puff) int {
	if p.Retries != nil {
		return *p.Retries
	}
	return d.Root.RetriesDefault
}

// execute runs one node's full attempt loop (with retries), then its
// self-handler if it ultimately failed, and records everything into
// Runtime State. It never returns an error — outcome is entirely
// reflected in state.PuffState.Status.
func (d *Dispatcher) execute(name string, node *graph.Node) {
	ps := d.State.Puffs[name]
	prof := d.resolveProfile(node.Puff.Profile)

	sem := d.pools[prof.Name]
	sem <- struct{}{}
	defer func() { <-sem }()

	d.mu.Lock()
	halted := d.halted
	d.mu.Unlock()
	if halted && !d.isExempt(node) {
		d.mu.Lock()
		ps.Status = state.Cancelled
		ps.Reason = "halt: never dispatched"
		ps.EndedAt = time.Now()
		d.mu.Unlock()
		return
	}

	maxAttempts := ps.MaxRetries + 1
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		rec := state.Attempt{Attempt: attempt, StartedAt: time.Now()}
		cancelled, err := d.runSteps(name, node, prof, attempt)
		rec.EndedAt = time.Now()
		if cancelled {
			rec.Status = state.Cancelled
			d.mu.Lock()
			ps.Attempts = append(ps.Attempts, rec)
			ps.Status = state.Cancelled
			ps.Reason = "halt: killed mid-execution"
			ps.EndedAt = time.Now()
			d.mu.Unlock()
			return
		}
		if err == nil {
			rec.Status = state.Success
			d.mu.Lock()
			ps.Attempts = append(ps.Attempts, rec)
			ps.Status = state.Success
			ps.EndedAt = time.Now()
			d.mu.Unlock()
			d.checkArtifacts(ps, node)
			return
		}
		rec.Status = state.Failed
		rec.Error = err.Error()
		d.mu.Lock()
		ps.Attempts = append(ps.Attempts, rec)
		d.mu.Unlock()
		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(backoffFunc(attempt))
		}
	}

	// exhausted retries
	d.mu.Lock()
	ps.Status = state.Failed
	d.mu.Unlock()
	if node.Puff.OnFailure != nil {
		fired := state.SelfHandlerResult{Fired: true}
		d.runHandlerSteps(name, node, prof)
		if node.Puff.OnFailure.Recover {
			fired.Recovered = true
			d.mu.Lock()
			ps.Status = state.Recovered
			d.mu.Unlock()
		}
		d.mu.Lock()
		ps.SelfHandler = &fired
		d.mu.Unlock()
	}
	d.mu.Lock()
	ps.EndedAt = time.Now()
	finalStatus := ps.Status
	d.mu.Unlock()
	d.checkArtifacts(ps, node)

	if finalStatus == state.Failed && d.Root.EffectiveOnFailure() == workspace.Halt {
		d.triggerHalt()
	}
	_ = lastErr
}

// backoffFunc computes retry backoff; a package variable so tests can
// shrink it instead of eating real wall-clock time for every retry
// case.
var backoffFunc = func(attempt int) time.Duration {
	return time.Duration(attempt) * 2 * time.Second
}

// triggerHalt stops new non-exempt dispatch and signals every
// currently running non-exempt process to terminate, with a grace
// period before a hard kill.
func (d *Dispatcher) triggerHalt() {
	d.mu.Lock()
	if d.halted {
		d.mu.Unlock()
		return
	}
	d.halted = true
	d.mu.Unlock()

	d.runningCmds.Range(func(key, value any) bool {
		name := key.(string)
		cmd := value.(*exec.Cmd)
		node := d.closure[name]
		if node != nil && d.isExempt(node) {
			return true
		}
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			go func(c *exec.Cmd) {
				time.Sleep(haltGrace)
				if c.ProcessState == nil {
					_ = c.Process.Kill()
				}
			}(cmd)
		}
		return true
	})
}

// checkArtifacts records whether every declared artifact path exists.
// This is drift/audit information only — never enforcement, since
// Mushmellow doesn't own the isolation boundary that would let it
// block a write.
func (d *Dispatcher) checkArtifacts(ps *state.PuffState, node *graph.Node) {
	var results []state.ArtifactResult
	for _, a := range node.Puff.Artifacts {
		path := a.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(d.Root.Dir, path)
		}
		_, err := os.Stat(path)
		results = append(results, state.ArtifactResult{
			Path:   a.Path,
			Exists: err == nil,
		})
	}
	if len(results) == 0 {
		return
	}
	d.mu.Lock()
	ps.Artifacts = append(ps.Artifacts, results...)
	d.mu.Unlock()
}

// runSteps executes every step of node in order. Returns
// cancelled=true if a halt killed the process mid-step.
func (d *Dispatcher) runSteps(name string, node *graph.Node, prof *puff.Profile, attempt int) (cancelled bool, err error) {
	env := buildEnv(prof, node.Puff.BranchOverrides)
	dir := prof.Cwd
	if dir == "" {
		dir = d.Root.Dir
	}
	shell := prof.Shell
	if shell == "" {
		shell = "sh"
	}

	var logBuf bytes.Buffer
	defer d.writeLog(name, attempt, &logBuf)

	for _, step := range node.Puff.Steps {
		switch step.Kind {
		case puff.StepMessage:
			fmt.Fprintln(&logBuf, step.Message)
			fmt.Println(step.Message)

		case puff.StepShell:
			cmd := exec.Command(shell, "-c", step.Command)
			cmd.Env = env
			cmd.Dir = dir
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf

			if err := cmd.Start(); err != nil {
				return false, fmt.Errorf("starting %q: %w", step.Command, err)
			}
			d.runningCmds.Store(name, cmd)
			waitErr := cmd.Wait()
			d.runningCmds.Delete(name)

			d.mu.Lock()
			halted := d.halted
			d.mu.Unlock()
			if waitErr != nil && halted && !d.isExempt(node) {
				return true, nil
			}
			if waitErr != nil {
				return false, fmt.Errorf("step %q: %w", step.Command, waitErr)
			}

		case puff.StepMember:
			if err := d.runMemberStep(name, step); err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

// runHandlerSteps runs a puff's on_failure block. Best-effort: a
// failure inside the handler itself is logged, not further escalated
// — designing nested handler-of-a-handler failure is out of scope
// for v1 and flagged as such.
func (d *Dispatcher) runHandlerSteps(name string, node *graph.Node, prof *puff.Profile) {
	env := buildEnv(prof, node.Puff.BranchOverrides)
	dir := prof.Cwd
	if dir == "" {
		dir = d.Root.Dir
	}
	shell := prof.Shell
	if shell == "" {
		shell = "sh"
	}
	var logBuf bytes.Buffer
	for _, step := range node.Puff.OnFailure.Steps {
		switch step.Kind {
		case puff.StepMessage:
			fmt.Println(step.Message)
		case puff.StepShell:
			cmd := exec.Command(shell, "-c", step.Command)
			cmd.Env = env
			cmd.Dir = dir
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf
			_ = cmd.Run() // best-effort, see doc comment
		case puff.StepMember:
			_ = d.runMemberStep(name, step)
		}
	}
	d.writeLog(name+"-handler", 1, &logBuf)
}

// runMemberStep delegates blocking and synchronously into a member
// workspace's own dependency closure. Members are self-contained: a
// nested Dispatcher and its own Runtime State, referenced (not
// inlined) from the parent's PuffState.MemberCalls.
func (d *Dispatcher) runMemberStep(parentPuff string, step puff.Step) error {
	memberDir := filepath.Join(d.Root.Dir, step.Member)
	memberRootPath, err := workspace.FindWorkspaceRoot(memberDir)
	if err != nil {
		// members are marked by *.mushmellow.yaml, not mushmellow.yaml;
		// look for that directly instead of walking upward, which
		// could otherwise escape into the parent workspace.
		memberRootPath = ""
	}
	if memberRootPath == "" {
		candidates, globErr := filepath.Glob(filepath.Join(memberDir, "*.mushmellow.yaml"))
		if globErr != nil || len(candidates) == 0 {
			return fmt.Errorf("member %q: no *.mushmellow.yaml found", step.Member)
		}
		memberRootPath = candidates[0]
	}

	memberRoot, err := workspace.ParseFile(memberRootPath)
	if err != nil {
		return fmt.Errorf("member %q: %w", step.Member, err)
	}
	memberRoot.IsMember = true

	memberGraph, err := graph.Build(memberRoot)
	if err != nil {
		return fmt.Errorf("member %q: building graph: %w", step.Member, err)
	}

	nested := New(memberRoot, memberGraph)
	nestedState, err := nested.Run(step.MemberPuff)
	if err != nil {
		return fmt.Errorf("member %q::%q: %w", step.Member, step.MemberPuff, err)
	}

	finalStatus := nestedState.Puffs[step.MemberPuff].Status

	d.mu.Lock()
	if ps, ok := d.State.Puffs[parentPuff]; ok {
		ps.MemberCalls = append(ps.MemberCalls, state.MemberCallRef{
			Member: step.Member,
			Puff:   step.MemberPuff,
			RunID:  nestedState.RunID,
			Status: finalStatus,
		})
	}
	d.mu.Unlock()

	if finalStatus != state.Success && finalStatus != state.Recovered {
		return fmt.Errorf("member %q::%q ended in status %q", step.Member, step.MemberPuff, finalStatus)
	}
	return nil
}

func (d *Dispatcher) writeLog(name string, attempt int, buf *bytes.Buffer) {
	path := state.LogPath(d.Root.Dir, d.State.RunID, name, attempt)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, buf.Bytes(), 0o644)
}

// buildEnv resolves a profile's effective environment: host env
// (unless InheritEnv is explicitly false) as the base, profile.Env
// overrides on top, then branch overrides (if any) on top of that.
func buildEnv(prof *puff.Profile, branchOverrides map[string]string) []string {
	merged := map[string]string{}
	if prof.InheritsEnv() {
		for _, kv := range os.Environ() {
			for i := 0; i < len(kv); i++ {
				if kv[i] == '=' {
					merged[kv[:i]] = kv[i+1:]
					break
				}
			}
		}
	}
	for k, v := range prof.Env {
		merged[k] = v
	}
	for k, v := range branchOverrides {
		merged[k] = v
	}
	out := make([]string, 0, len(merged))
	for k, v := range merged {
		out = append(out, k+"="+v)
	}
	return out
}
