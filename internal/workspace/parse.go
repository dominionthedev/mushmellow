package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// rootFilenames are the accepted names for a workspace root file, in
// lookup priority order.
var rootFilenames = []string{"mushmellow.yaml", "mushmellow.yml"}

// ParseFile reads and validates a single mushmellow.yaml (or member
// file) from disk. It does not discover members — call
// DiscoverMembers separately from the workspace root.
func ParseFile(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var r Root
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	r.Dir = filepath.Dir(path)

	if err := r.finalize(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &r, nil
}

// FindWorkspaceRoot walks upward from startDir looking for a
// mushmellow.yaml/.yml, matching the definition: "the workspace is
// where mushmellow.yml/yaml exists."
func FindWorkspaceRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", startDir, err)
	}
	for {
		for _, name := range rootFilenames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no mushmellow.yaml found above %s", startDir)
		}
		dir = parent
	}
}

// DiscoverMembers walks the tree beneath the workspace root looking
// for *.mushmellow.yaml files (member markers). It stops descending
// into .mushmellow/, .git/, and any directory that is itself a member
// root, so nested members are found but not double-walked past their
// own boundary.
//
// Returns member Roots parsed and finalized, keyed by their
// folder-relative path from the workspace root (e.g. "cmd/api").
func DiscoverMembers(rootDir string) (map[string]*Root, error) {
	members := make(map[string]*Root)

	skipDirs := map[string]bool{
		".git":        true,
		".mushmellow": true,
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != rootDir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if path == filepath.Join(rootDir, "mushmellow.yaml") || path == filepath.Join(rootDir, "mushmellow.yml") {
			return nil // that's the root itself, not a member
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".mushmellow.yaml") && !strings.HasSuffix(name, ".mushmellow.yml") {
			return nil
		}

		memberDir := filepath.Dir(path)
		rel, err := filepath.Rel(rootDir, memberDir)
		if err != nil {
			return fmt.Errorf("computing relative path for member %s: %w", path, err)
		}

		r, err := ParseFile(path)
		if err != nil {
			return fmt.Errorf("member %s: %w", rel, err)
		}
		r.IsMember = true
		members[rel] = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return members, nil
}
