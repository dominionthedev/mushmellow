package config

// Config represents the root mushmellow.yaml file
type Config struct {
	Version     int                   `yaml:"version"`
	Name        string                `yaml:"name"`
	Meta        Meta                  `yaml:"meta,omitempty"`
	Env         map[string]string     `yaml:"env,omitempty"`
	CI          CIConfig              `yaml:"ci,omitempty"`
	Mushmellows map[string]Mushmellow `yaml:"mushmellows"`
}

// Meta contains optional metadata
type Meta struct {
	Author      string `yaml:"author,omitempty"`
	Description string `yaml:"description,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
}

// CIConfig defines CI compatibility settings
type CIConfig struct {
	Compatible bool   `yaml:"compatible"`
	Output     string `yaml:"output,omitempty"` // junit, json, text
	Strict     bool   `yaml:"strict"`
}

// Mushmellow is a named workflow
type Mushmellow struct {
	Description string            `yaml:"description,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Puffs       []Puff            `yaml:"puffs"`
}

// Puff is the atomic execution unit
type Puff struct {
	ID        string            `yaml:"id"`
	Name      string            `yaml:"name,omitempty"`
	Type      string            `yaml:"type,omitempty"` // run, message, wait
	Run       string            `yaml:"run,omitempty"`
	Text      string            `yaml:"text,omitempty"`     // for message type
	Duration  string            `yaml:"duration,omitempty"` // for wait type
	Dir       string            `yaml:"dir,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Timeout   string            `yaml:"timeout,omitempty"`
	DependsOn []string          `yaml:"depends_on,omitempty"`
}
