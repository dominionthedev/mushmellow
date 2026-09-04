// Package puff defines Mushmellow's core domain types: the smallest
// orchestration primitive (Puff), its execution environment (Profile),
// and the pieces that compose a puff's behavior (Step, DependsOn,
// BranchSpec, OnFailure, Criterion).
//
// This package holds data shapes only. No scheduling, no execution,
// no I/O. Those live in internal/graph and internal/scheduler.
package puff

import "fmt"

// EdgeCondition controls when a dependency edge makes its downstream
// puff eligible to run relative to the upstream puff's outcome.
type EdgeCondition string

const (
	OnSuccess     EdgeCondition = "success" // default when unspecified
	OnFailureCond EdgeCondition = "failure"
	OnAlways      EdgeCondition = "always"
)

// DependsOn is one edge in the dependency graph: this puff becomes a
// candidate for readiness once the named upstream puff reaches a
// terminal state matching Condition.
type DependsOn struct {
	Puff      string        `yaml:"puff"`
	Condition EdgeCondition `yaml:"on,omitempty"`
}

// StepKind distinguishes the three things a step can do. Message is
// deliberately not a Puff type — see Puff.Type doc comment — it is
// only ever a step.
type StepKind string

const (
	StepShell   StepKind = "shell"   // run a command in the assigned Profile's shell
	StepMessage StepKind = "message" // print a string; cannot fail
	StepMember  StepKind = "member"  // blocking, synchronous call into a member's puff
)

// Step is one unit inside a Puff's execution body. Steps run in
// declared order. A failing StepShell or StepMember step fails the
// containing Puff; a StepMessage step never fails.
//
// YAML shape (exactly one key per step):
//
//   - run: cargo build
//   - echo: "building api"
//   - member: cmd/api::build
type Step struct {
	Kind StepKind

	// StepShell
	Command string

	// StepMessage
	Message string

	// StepMember: "cmd/api::build" — Member is the folder-relative
	// path to the member workspace, MemberPuff is the puff name
	// inside it. Blocking and synchronous: the containing puff does
	// not continue until the member puff's own dependency closure
	// finishes.
	Member     string
	MemberPuff string
}

// CriterionKind names the external, world-facing preconditions a puff
// can require before it is eligible to dispatch. These are distinct
// from DependsOn (puff-to-puff) and Artifacts (post-run drift
// detection) — a Criterion checks something outside Mushmellow's own
// run state.
type CriterionKind string

const (
	CriterionFileContains CriterionKind = "file_contains"
	CriterionEnvSet       CriterionKind = "env_set"
	CriterionEnvEquals    CriterionKind = "env_equals"
	CriterionCommandOK    CriterionKind = "command_ok"
	CriterionPortOpen     CriterionKind = "port_open"
)

// Criterion is one external precondition. Evaluated at dispatch time
// (not at parse time), because the world can change mid-run — see
// design notes on staleness.
type Criterion struct {
	Kind CriterionKind `yaml:"kind"`

	// CriterionFileContains
	Path   string `yaml:"path,omitempty"`
	Substr string `yaml:"contains,omitempty"`

	// CriterionEnvSet / CriterionEnvEquals
	EnvVar   string `yaml:"env,omitempty"`
	EnvValue string `yaml:"equals,omitempty"`

	// CriterionCommandOK
	Command string `yaml:"command,omitempty"`

	// CriterionPortOpen
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

// ArtifactDecl is a puff's declared-intent output path. Mushmellow
// indexes it and flags drift (writes outside every declared path) as
// a non-blocking diagnostic. This is an audit trail, not access
// control — Mushmellow does not and cannot enforce it, because
// isolation is external (Runbox/Docker), not something Mushmellow
// owns.
type ArtifactDecl struct {
	Path string `yaml:"path"`
	Copy bool   `yaml:"copy,omitempty"` // snapshot into .mushmellow/, default false (reference-only)
}

// OnFailure is a puff-scoped self-handler: a set of steps that run
// when the puff's own steps fail. Firing the handler never changes
// the puff's terminal status by itself — status only becomes
// "recovered" if Recover is explicitly true. No implicit swallowing.
type OnFailure struct {
	Steps   []Step `yaml:"steps"`
	Recover bool   `yaml:"recover,omitempty"`
}

// BranchSpec declares parameterized variance for a puff: it is
// expanded into sibling puffs before scheduling via ExpandPuff, never
// as a runtime graph mutation. Overrides is a field name -> list of
// values; expansion is the cartesian product, but v0.1 only needs
// single-field matrices in practice.
type BranchSpec struct {
	Overrides map[string][]string `yaml:"matrix"`
}

// Puff is the smallest orchestration primitive. Type is deliberately
// fixed to "run" for v0.1 — "message" and "service" were considered
// and cut: message has no independent failure mode (it's a Step, not
// a Puff type) and service needs a readiness-predicate + crash
// detection design that doesn't exist yet.
type Puff struct {
	Name string `yaml:"-"` // set from the map key during parse

	Steps     []Step      `yaml:"steps"`
	DependsOn []DependsOn `yaml:"depends_on,omitempty"`
	When      []Criterion `yaml:"when,omitempty"`

	Profile string `yaml:"profile,omitempty"` // empty -> falls to base "mushmellow" profile
	Pool    *int   `yaml:"pool,omitempty"`    // puff-level override, top of precedence chain
	Retries *int   `yaml:"retries,omitempty"` // puff-level override of workspace default

	Artifacts []ArtifactDecl `yaml:"artifacts,omitempty"`
	OnFailure *OnFailure     `yaml:"on_failure,omitempty"`
	Branch    *BranchSpec    `yaml:"branch,omitempty"`

	// BranchOverrides is set on expanded (post-branch) puffs only —
	// never present in raw YAML. It records the field->value
	// overrides this variant was expanded with, applied as env
	// overrides layered onto the resolved profile's env at
	// execution time. BranchName/Ephemeral travel alongside it for
	// vault naming (puff@branch or puff@branch~runidhash).
	BranchOverrides map[string]string `yaml:"-"`
	BranchName      string            `yaml:"-"`
	BranchEphemeral bool              `yaml:"-"`
}

// Validate checks structural invariants that are cheap to catch at
// parse time, before this puff ever reaches graph construction.
func (p *Puff) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("puff has empty name")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("puff %q has no steps", p.Name)
	}
	for i, d := range p.DependsOn {
		if d.Puff == "" {
			return fmt.Errorf("puff %q: depends_on[%d] has empty puff name", p.Name, i)
		}
		switch d.Condition {
		case "", OnSuccess, OnFailureCond, OnAlways:
		default:
			return fmt.Errorf("puff %q: depends_on[%d] has invalid condition %q", p.Name, i, d.Condition)
		}
	}
	if p.Pool != nil && *p.Pool < 1 {
		return fmt.Errorf("puff %q: pool override must be >= 1", p.Name)
	}
	if p.Retries != nil && *p.Retries < 0 {
		return fmt.Errorf("puff %q: retries override must be >= 0", p.Name)
	}
	return nil
}

// EffectiveCondition returns the edge condition, defaulting to
// OnSuccess per the locked rule: "on" is implicit success when
// unspecified.
func (d DependsOn) EffectiveCondition() EdgeCondition {
	if d.Condition == "" {
		return OnSuccess
	}
	return d.Condition
}

// Profile defines where and with what resources a puff executes.
// Execution-type (Local/Isolated) was designed and killed: isolation
// is achieved by running Mushmellow itself inside Runbox/Docker, not
// by Mushmellow orchestrating a backend. A Profile is deliberately
// just: environment, shell, working directory, and a concurrency
// budget.
type Profile struct {
	Name string `yaml:"-"`

	Env   map[string]string `yaml:"env,omitempty"`
	Shell string            `yaml:"shell,omitempty"`
	Cwd   string            `yaml:"cwd,omitempty"`

	// Pool is this profile's worker concurrency budget. Enforced
	// per-profile, not globally — two profiles each with pool=4 can
	// run 8 processes concurrently in total. That's an accepted
	// tradeoff (composability over a global ceiling), not an
	// oversight.
	Pool int `yaml:"pool,omitempty"`

	// InheritEnv controls whether the host shell's environment is
	// inherited before Env overrides are applied. Defaults to true
	// (standard practice, matches Make/Just/Taskfile) — set false
	// for a from-scratch env.
	InheritEnv *bool `yaml:"inherit_env,omitempty"`
}

// InheritsEnv reports the effective default: inherit unless
// explicitly disabled.
func (p Profile) InheritsEnv() bool {
	return p.InheritEnv == nil || *p.InheritEnv
}

// BaseProfileName is the implicit profile every puff without an
// explicit Profile assignment falls back to. Its default pool is 1
// (serial, safe default) unless the workspace overrides it.
const BaseProfileName = "mushmellow"

// DefaultPool is the pool size used when a profile declares none.
const DefaultPool = 1

// ResolvePool implements the locked precedence chain: puff-level
// override -> assigned profile's pool -> base "mushmellow" profile
// default.
func ResolvePool(p *Puff, assigned *Profile, base *Profile) int {
	if p.Pool != nil {
		return *p.Pool
	}
	if assigned != nil && assigned.Pool > 0 {
		return assigned.Pool
	}
	if base != nil && base.Pool > 0 {
		return base.Pool
	}
	return DefaultPool
}
