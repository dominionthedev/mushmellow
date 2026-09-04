package puff

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML decodes a step's one-of shape: exactly one of
// run/echo/member must be present. member is further split into
// Member (folder path) and MemberPuff (puff name inside it) on the
// "::" separator.
func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Run    string `yaml:"run"`
		Echo   string `yaml:"echo"`
		Member string `yaml:"member"`
	}
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("decoding step: %w", err)
	}

	set := 0
	if raw.Run != "" {
		set++
	}
	if raw.Echo != "" {
		set++
	}
	if raw.Member != "" {
		set++
	}
	if set == 0 {
		return fmt.Errorf("step has none of run/echo/member set")
	}
	if set > 1 {
		return fmt.Errorf("step must set exactly one of run/echo/member, got %d", set)
	}

	switch {
	case raw.Run != "":
		s.Kind = StepShell
		s.Command = raw.Run
	case raw.Echo != "":
		s.Kind = StepMessage
		s.Message = raw.Echo
	case raw.Member != "":
		parts := strings.SplitN(raw.Member, "::", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("step member %q must be in form <path>::<puff>", raw.Member)
		}
		s.Kind = StepMember
		s.Member = parts[0]
		s.MemberPuff = parts[1]
	}
	return nil
}
