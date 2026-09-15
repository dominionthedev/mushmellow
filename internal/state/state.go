// Package state defines Mushmellow's Runtime State: the persisted
// record of one invocation, written to
// .mushmellow/runs/<run_id>/state.json. This is the single source of
// truth `mushmellow inspect` reads from — Diagnostics is a formatted
// view over this, not a separate subsystem.
package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Status is one of six terminal states a puff can end in, plus two
// transient in-flight states.
type Status string

const (
	Pending   Status = "pending"
	Running   Status = "running"
	Success   Status = "success"
	Failed    Status = "failed"
	Recovered Status = "recovered" // self-handler explicitly recovered
	Blocked   Status = "blocked"   // an actual failure occurred upstream
	// (this puff's own execution, or an upstream puff's) and this
	// puff can never satisfy its dependency edges as a result.
	Skipped Status = "skipped" // this puff never ran, but nothing
	// failed: an edge condition can never fire (an on:"failure"
	// dependent whose watched puff succeeded), a when: precondition
	// wasn't met, or an upstream puff was itself Skipped. Distinct
	// from Blocked on purpose - "the world wasn't in the right
	// state" and "something broke" are different situations and a
	// report that conflates them is misleading.
	Cancelled Status = "cancelled" // halt mode killed it mid-flight,
	// or it was never dispatched because halt was already in effect.
)

// IsTerminal reports whether a status will never change again.
func (s Status) IsTerminal() bool {
	switch s {
	case Success, Failed, Recovered, Blocked, Skipped, Cancelled:
		return true
	default:
		return false
	}
}

// SatisfiesSuccess reports whether this terminal status counts as
// "success" for a downstream on:"success" edge. Recovered counts —
// a self-handler that explicitly recovered lets normal downstream
// proceed, per the locked failure model.
func (s Status) SatisfiesSuccess() bool {
	return s == Success || s == Recovered
}

// SatisfiesFailure reports whether this terminal status counts as
// "failure" for a downstream on:"failure" edge. Recovered counts —
// the puff did fail (attempts exhausted) before its self-handler
// recovered it, and an external on:"failure" watcher exists
// specifically to react to failures even ones that get handled
// internally too. Blocked/Skipped/Cancelled do not count: none of
// them mean *this* puff failed, they mean it never ran.
func (s Status) SatisfiesFailure() bool {
	return s == Failed || s == Recovered
}

// CascadeStatus decides what a downstream on:"success" dependent
// becomes when upstream didn't satisfy success. Failed or Blocked
// upstream means a real failure happened somewhere in the chain -
// Blocked propagates as Blocked. Skipped/Cancelled upstream means
// nothing failed - the downstream inherits the same "didn't happen"
// reason instead of being mislabeled as a failure.
func CascadeStatus(upstream Status) Status {
	switch upstream {
	case Failed, Blocked:
		return Blocked
	case Cancelled:
		return Cancelled
	default: // Skipped, or any unexpected terminal status
		return Skipped
	}
}

// Attempt is one execution attempt of a puff's steps. A puff with
// retries has multiple attempts recorded; the last one determines
// the puff's final status.
type Attempt struct {
	Attempt   int       `json:"attempt"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// SelfHandlerResult records whether a puff's on_failure block ran and
// whether it explicitly recovered. Firing never implies recovery —
// no swallow-by-default.
type SelfHandlerResult struct {
	Fired     bool `json:"fired"`
	Recovered bool `json:"recovered"`
}

// MemberCallRef points at a nested member run's own state file,
// rather than inlining it — members are self-contained workspaces
// with their own run history, independent of who called them.
type MemberCallRef struct {
	Member string `json:"member"`
	Puff   string `json:"puff"`
	RunID  string `json:"run_id"`
	Status Status `json:"status"`
}

// ArtifactResult is the recorded outcome of one declared artifact
// path: whether it exists, and whether anything changed outside
// every declared path (drift is a diagnostic, never enforcement).
type ArtifactResult struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// PuffState is one puff's full record within a run.
type PuffState struct {
	Status     Status    `json:"status"`
	Attempts   []Attempt `json:"attempts"`
	MaxRetries int       `json:"max_retries"`
	// Reason is a human-readable explanation, set whenever Status is
	// Blocked, Skipped, or Cancelled. For Blocked it's usually an
	// upstream puff name; for Skipped it describes the unmet edge
	// condition or when: criterion; for Cancelled it names what
	// triggered the halt. Never set for Success/Failed/Recovered -
	// those are self-explanatory from Attempts.
	Reason       string             `json:"reason,omitempty"`
	Profile      string             `json:"profile"`
	ResolvedPool int                `json:"resolved_pool"`
	Branch       string             `json:"branch,omitempty"`
	SelfHandler  *SelfHandlerResult `json:"self_handler,omitempty"`
	MemberCalls  []MemberCallRef    `json:"member_calls,omitempty"`
	Artifacts    []ArtifactResult   `json:"artifacts,omitempty"`
	StartedAt    time.Time          `json:"started_at,omitempty"`
	EndedAt      time.Time          `json:"ended_at,omitempty"`
}

// Run is the full Runtime State for one invocation.
type Run struct {
	RunID         string                `json:"run_id"`
	InvokedPuff   string                `json:"invoked_puff"`
	Workspace     string                `json:"workspace"`
	OnFailureMode string                `json:"on_failure_mode"`
	StartedAt     time.Time             `json:"started_at"`
	EndedAt       time.Time             `json:"ended_at,omitempty"`
	Puffs         map[string]*PuffState `json:"puffs"`
	Cancelled     []string              `json:"cancelled,omitempty"`
}

// NewRunID generates a short, filesystem-safe run identifier.
//
// Bug fixed here: an earlier version took the leading hex digits of
// UnixNano(), which barely change between calls microseconds apart -
// exactly the digits least likely to differ for two runs invoked back
// to back. Two runs seconds apart collided on the same ID as a
// result. Random bytes have no such structure to accidentally rely on.
func NewRunID() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is effectively unrecoverable on any real
		// system; fall back to a timestamp-based ID rather than a
		// zero-value one, which would silently collide on every call.
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Save writes the run state to
// <workspaceDir>/.mushmellow/runs/<run_id>/state.json.
func (r *Run) Save(workspaceDir string) error {
	dir := filepath.Join(workspaceDir, ".mushmellow", "runs", r.RunID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating run dir: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling run state: %w", err)
	}
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Load reads a previously saved run state by ID.
func Load(workspaceDir, runID string) (*Run, error) {
	path := filepath.Join(workspaceDir, ".mushmellow", "runs", runID, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var r Run
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &r, nil
}

// LogPath returns where a given attempt's captured stdout/stderr
// should be written.
func LogPath(workspaceDir, runID, puffName string, attempt int) string {
	dir := filepath.Join(workspaceDir, ".mushmellow", "runs", runID, "logs")
	safe := sanitizeForFilename(puffName)
	return filepath.Join(dir, fmt.Sprintf("%s-attempt%d.log", safe, attempt))
}

func sanitizeForFilename(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
