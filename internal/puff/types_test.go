package puff

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStepUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string // substring, empty means no error
		check   func(t *testing.T, s Step)
	}{
		{
			name: "run",
			yaml: `run: cargo build`,
			check: func(t *testing.T, s Step) {
				if s.Kind != StepShell || s.Command != "cargo build" {
					t.Errorf("got %+v", s)
				}
			},
		},
		{
			name: "echo",
			yaml: `echo: "building api"`,
			check: func(t *testing.T, s Step) {
				if s.Kind != StepMessage || s.Message != "building api" {
					t.Errorf("got %+v", s)
				}
			},
		},
		{
			name: "member",
			yaml: `member: cmd/api::build`,
			check: func(t *testing.T, s Step) {
				if s.Kind != StepMember || s.Member != "cmd/api" || s.MemberPuff != "build" {
					t.Errorf("got %+v", s)
				}
			},
		},
		{
			name:    "member missing separator",
			yaml:    `member: cmdapi`,
			wantErr: "form",
		},
		{
			name:    "empty step",
			yaml:    `{}`,
			wantErr: "none of run/echo/member",
		},
		{
			name:    "two keys set",
			yaml:    "run: cargo build\necho: hi",
			wantErr: "exactly one",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s Step
			err := yaml.Unmarshal([]byte(tc.yaml), &s)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, s)
		})
	}
}

func TestPuffValidate(t *testing.T) {
	cases := []struct {
		name    string
		puff    Puff
		wantErr string
	}{
		{
			name: "valid",
			puff: Puff{
				Name:  "build",
				Steps: []Step{{Kind: StepShell, Command: "go build"}},
			},
		},
		{
			name:    "no steps",
			puff:    Puff{Name: "empty"},
			wantErr: "no steps",
		},
		{
			name: "bad depends_on condition",
			puff: Puff{
				Name:      "b",
				Steps:     []Step{{Kind: StepShell, Command: "x"}},
				DependsOn: []DependsOn{{Puff: "a", Condition: "sometimes"}},
			},
			wantErr: "invalid condition",
		},
		{
			name: "zero pool override",
			puff: Puff{
				Name:  "b",
				Steps: []Step{{Kind: StepShell, Command: "x"}},
				Pool:  intPtr(0),
			},
			wantErr: "pool override",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.puff.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestResolvePool_Precedence(t *testing.T) {
	base := &Profile{Name: BaseProfileName, Pool: DefaultPool}
	assigned := &Profile{Name: "build", Pool: 4}

	// puff override wins over everything
	p := &Puff{Pool: intPtr(8)}
	if got := ResolvePool(p, assigned, base); got != 8 {
		t.Errorf("want 8, got %d", got)
	}

	// assigned profile wins when no puff override
	p = &Puff{}
	if got := ResolvePool(p, assigned, base); got != 4 {
		t.Errorf("want 4, got %d", got)
	}

	// falls to base when no assignment at all
	if got := ResolvePool(p, nil, base); got != DefaultPool {
		t.Errorf("want %d, got %d", DefaultPool, got)
	}
}

func TestCriterionValidate(t *testing.T) {
	cases := []struct {
		name    string
		c       Criterion
		wantErr string
	}{
		{name: "file_contains valid", c: Criterion{Kind: CriterionFileContains, Path: "x", Substr: "y"}},
		{name: "file_contains missing path", c: Criterion{Kind: CriterionFileContains, Substr: "y"}, wantErr: "requires path"},
		{name: "file_contains missing contains", c: Criterion{Kind: CriterionFileContains, Path: "x"}, wantErr: "requires contains"},
		{name: "env_set valid", c: Criterion{Kind: CriterionEnvSet, EnvVar: "X"}},
		{name: "env_set missing env", c: Criterion{Kind: CriterionEnvSet}, wantErr: "requires env"},
		{name: "env_equals valid", c: Criterion{Kind: CriterionEnvEquals, EnvVar: "X", EnvValue: "y"}},
		{name: "env_equals missing value", c: Criterion{Kind: CriterionEnvEquals, EnvVar: "X"}, wantErr: "requires equals"},
		{name: "command_ok valid", c: Criterion{Kind: CriterionCommandOK, Command: "true"}},
		{name: "command_ok missing command", c: Criterion{Kind: CriterionCommandOK}, wantErr: "requires command"},
		{name: "port_open valid", c: Criterion{Kind: CriterionPortOpen, Host: "localhost", Port: 8080}},
		{name: "port_open missing host", c: Criterion{Kind: CriterionPortOpen, Port: 8080}, wantErr: "requires host"},
		{name: "port_open bad port", c: Criterion{Kind: CriterionPortOpen, Host: "localhost", Port: 70000}, wantErr: "valid port"},
		{name: "unknown kind", c: Criterion{Kind: "carrier_pigeon"}, wantErr: "unknown criterion kind"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func intPtr(i int) *int { return &i }
