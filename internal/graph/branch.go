package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dominionthedev/mushmellow/internal/puff"
)

// ExpandPuff clones base and applies overrides as a named branch
// variant. This is the single mechanism behind both static
// (parse-time, matrix-declared) and on-demand (CLI-driven, ephemeral)
// branching — they must never diverge into separate code paths.
func ExpandPuff(base puff.Puff, branchName string, overrides map[string]string, ephemeral bool) puff.Puff {
	clone := base
	clone.Name = fmt.Sprintf("%s@%s", base.Name, branchName)
	clone.Branch = nil // expanded puffs are concrete, not templates
	clone.BranchOverrides = overrides
	clone.BranchName = branchName
	clone.BranchEphemeral = ephemeral

	// DependsOn is a slice; copy it so mutating one variant's edges
	// (during cascade rewriting) never aliases another variant's.
	if base.DependsOn != nil {
		clone.DependsOn = append([]puff.DependsOn(nil), base.DependsOn...)
	}
	return clone
}

// branchCombo is one expansion point: a name plus its concrete field
// overrides, e.g. "env-A" -> {"CARGO_TARGET_DIR": ".../env-A"}.
type branchCombo struct {
	name      string
	overrides map[string]string
}

// expandMatrix turns a BranchSpec (field -> list of values) into the
// cartesian product of combos, each with a deterministic, readable
// branch name. v0.1 note: a single-field matrix (the common case —
// one puff, one overridden field, e.g. env) is fully supported.
// Multi-field matrices produce the full cartesian product but the
// resulting name is a join of every field=value pair, which gets
// unwieldy past two fields — acceptable for now, revisit if a real
// workflow needs three+ dimensions.
func expandMatrix(spec *puff.BranchSpec) []branchCombo {
	if spec == nil || len(spec.Overrides) == 0 {
		return nil
	}

	fields := make([]string, 0, len(spec.Overrides))
	for f := range spec.Overrides {
		fields = append(fields, f)
	}
	sort.Strings(fields) // deterministic order -> deterministic names

	combos := []branchCombo{{name: "", overrides: map[string]string{}}}
	for _, field := range fields {
		values := spec.Overrides[field]
		var next []branchCombo
		for _, c := range combos {
			for _, v := range values {
				ov := make(map[string]string, len(c.overrides)+1)
				for k, val := range c.overrides {
					ov[k] = val
				}
				ov[field] = v
				name := v
				if c.name != "" {
					name = c.name + "-" + v
				}
				next = append(next, branchCombo{name: name, overrides: ov})
			}
		}
		combos = next
	}

	for i := range combos {
		combos[i].name = sanitizeBranchName(combos[i].name)
	}
	return combos
}

// sanitizeBranchName makes a matrix value safe as a branch label and,
// by extension, a vault key segment (puff@branch). Values containing
// path separators (a real risk — matrix values are often paths, as
// in the CARGO_TARGET_DIR case) are flattened rather than left to
// silently produce a name with embedded "/".
func sanitizeBranchName(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer(
		" ", "-",
		"_", "-",
		"/", "-",
		"\\", "-",
	).Replace(s)
	return s
}
