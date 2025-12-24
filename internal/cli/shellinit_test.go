package cli

import (
	"strings"
	"testing"
)

func TestBuildShellInitScript(t *testing.T) {
	tests := []struct {
		name   string
		shell  string
		substr string
	}{
		{"fish", "fish", "GW_CALLER_CWD"},
		{"bash", "bash", "command gw"},
		{"default", "", "function gw"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := buildShellInitScript(tt.shell)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(script, tt.substr) {
				t.Fatalf("script missing substring %q", tt.substr)
			}
		})
	}
}

func TestBuildShellInitScriptUnsupported(t *testing.T) {
	if _, err := buildShellInitScript("tcsh"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}

func TestShellInitContainsNavigationCommands(t *testing.T) {
	fish, err := buildShellInitScript("fish")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fish, "case go new add mv") {
		t.Fatalf("fish script should handle navigation commands")
	}

	bash, err := buildShellInitScript("bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(bash, "go|new|add|mv") {
		t.Fatalf("bash script should handle navigation commands")
	}
}
