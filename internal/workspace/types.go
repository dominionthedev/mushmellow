// Package workspace represents a parsed mushmellow.yaml (or a
// member's *.mushmellow.yaml) and the discovery of members beneath a
// workspace root. Parsing only — no graph construction, no
// scheduling.
package workspace

import (
	"fmt"

	"github.com/dominionthedev/mushmellow/internal/puff"
)

// OnFailureMode is the workspace-level, closure-scoped failure
// posture. Scoped to the invoked puff's dependency closure — there
// is no "whole workflow" object to halt, since Mushmellow only ever
// executes a puff and whatever its dependencies pull in.
type OnFailureMode string

const (
	// Isolate (default): a failed puff blocks only its own
	// downstream subtree (reachability from the failure point).
	// Everything outside that subtree keeps running untouched.
	Isolate OnFailureMode = "isolate"

	// Halt: on first unrecovered failure (after retries exhausted),
	// stop dispatching new puffs and cancel every in-flight process
	// within the invoked closure, except puffs whose readiness
	// condition is on:"failure" or on:"always" relative to the
	// failure — those are allowed to finish so cleanup/notify can
	// run.
	Halt OnFailureMode = "halt"
)

// Root is a parsed mushmellow.yaml — either the workspace root or a
// member (*.mushmellow.yaml in a subfolder). Members are
// folder-scoped and independently invokable: there are no
// cross-member puff dependency edges. A root puff calls into a
// member only via a StepMember step, which is a blocking synchronous
// invocation of the member's own dependency closure.
type Root struct {
	// Dir is the absolute path to the folder containing this file.
	Dir string `yaml:"-"`

	// IsMember is true when this Root was discovered as
	// *.mushmellow.yaml under a workspace root, false for the
	// top-level mushmellow.yaml.
	IsMember bool `yaml:"-"`

	OnFailure      OnFailureMode `yaml:"on_failure,omitempty"`
	RetriesDefault int           `yaml:"retries,omitempty"`

	Profiles map[string]puff.Profile `yaml:"profiles,omitempty"`
	Puffs    map[string]puff.Puff    `yaml:"puffs"`

	// Context holds arbitrary workspace-level config values.
	// Global (~/.mushmellow/context.yaml) and workspace context
	// merge by override-by-key: workspace wins per key, no deep
	// merge.
	Context map[string]string `yaml:"context,omitempty"`
}

// EffectiveOnFailure returns the configured mode, defaulting to
// Isolate.
func (r *Root) EffectiveOnFailure() OnFailureMode {
	if r.OnFailure == "" {
		return Isolate
	}
	return r.OnFailure
}

// finalize sets derived fields (Name from map key) and validates
// every puff/profile after raw YAML decode.
func (r *Root) finalize() error {
	switch r.OnFailure {
	case "", Isolate, Halt:
	default:
		return fmt.Errorf("invalid on_failure mode %q (want %q or %q)", r.OnFailure, Isolate, Halt)
	}
	if r.RetriesDefault < 0 {
		return fmt.Errorf("retries default must be >= 0")
	}

	for name, prof := range r.Profiles {
		prof.Name = name
		if prof.Pool < 0 {
			return fmt.Errorf("profile %q: pool must be >= 0", name)
		}
		r.Profiles[name] = prof
	}

	if len(r.Puffs) == 0 {
		return fmt.Errorf("workspace defines no puffs")
	}
	for name, p := range r.Puffs {
		p.Name = name
		if err := p.Validate(); err != nil {
			return err
		}
		if p.Profile != "" {
			if _, ok := r.Profiles[p.Profile]; !ok {
				return fmt.Errorf("puff %q references unknown profile %q", name, p.Profile)
			}
		}
		r.Puffs[name] = p
	}
	return nil
}

// BaseProfile returns the implicit "mushmellow" profile, synthesizing
// a default (pool=1, inherit env) if the workspace didn't declare
// one explicitly.
func (r *Root) BaseProfile() puff.Profile {
	if p, ok := r.Profiles[puff.BaseProfileName]; ok {
		return p
	}
	return puff.Profile{
		Name: puff.BaseProfileName,
		Pool: puff.DefaultPool,
	}
}
