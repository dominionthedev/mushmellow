package config

import (
	"bufio"
	"fmt"
	"os"
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

// LoadDefault searches for mushmellow.yaml in common locations
func LoadDefault() (*Config, error) {
	searchPaths := []string{
		"mushmellow.yaml",
		"mushmellow.yml",
		".mushmellow.yaml",
	}

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			return Load(p)
		}
	}

	return nil, fmt.Errorf("mushmellow.yaml not found in current directory")
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
