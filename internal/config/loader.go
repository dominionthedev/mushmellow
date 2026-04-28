package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadEnv reads a .env file and returns its content as a map
func LoadEnv(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return env, scanner.Err()
}

// Load reads and parses the mushmellow configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadDefault searches for and merges all discovered mushmellow configuration files
func LoadDefault() (*Config, error) {
	// Root configs to check first
	roots := []string{
		"mushmellow.yaml",
		"mushmellow.yml",
		".mushmellow.yaml",
	}

	var rootCfg *Config

	for _, p := range roots {
		if _, err := os.Stat(p); err == nil {
			cfg, err := Load(p)
			if err == nil {
				rootCfg = cfg
				break
			}
		}
	}

	if rootCfg == nil {
		// If no root config, check for any *.mushmellow.yaml
		matches, _ := filepath.Glob("*.mushmellow.yaml")
		if len(matches) == 0 {
			return nil, fmt.Errorf("no mushmellow.yaml or *.mushmellow.yaml found")
		}
		
		// Load first match as base if no root found
		cfg, err := Load(matches[0])
		if err != nil {
			return nil, err
		}
		rootCfg = cfg
		matches = matches[1:] // Remove first as it's already base

		// Merge others
		for _, m := range matches {
			other, err := Load(m)
			if err == nil {
				mergeConfig(rootCfg, other)
			}
		}
	} else {
		// If root found, also look for and merge any *.mushmellow.yaml
		matches, _ := filepath.Glob("*.mushmellow.yaml")
		for _, m := range matches {
			// Skip if it's one of our root files
			isRoot := false
			for _, r := range roots {
				if m == r {
					isRoot = true
					break
				}
			}
			if isRoot {
				continue
			}

			other, err := Load(m)
			if err == nil {
				mergeConfig(rootCfg, other)
			}
		}
	}

	return rootCfg, nil
}

func mergeConfig(base, other *Config) {
	if base.Mushmellows == nil {
		base.Mushmellows = make(map[string]Mushmellow)
	}
	for name, m := range other.Mushmellows {
		base.Mushmellows[name] = m
	}
	// Also merge global env if needed
	if other.Env != nil {
		if base.Env == nil {
			base.Env = make(map[string]string)
		}
		for k, v := range other.Env {
			base.Env[k] = v
		}
	}
}

// Validate checks the configuration for required fields and basic correctness
func (c *Config) Validate() error {
	if c.Version == 0 {
		return fmt.Errorf("version is required")
	}
	if c.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(c.Mushmellows) == 0 {
		return fmt.Errorf("at least one mushmellow must be defined")
	}

	for name, m := range c.Mushmellows {
		if len(m.Puffs) == 0 {
			return fmt.Errorf("mushmellow '%s' has no puffs", name)
		}

		puffIDs := make(map[string]bool)
		for _, p := range m.Puffs {
			if p.ID == "" {
				return fmt.Errorf("puff in mushmellow '%s' is missing ID", name)
			}
			if puffIDs[p.ID] {
				return fmt.Errorf("duplicate puff ID '%s' in mushmellow '%s'", p.ID, name)
			}
			puffIDs[p.ID] = true
		}
	}

	return nil
}
