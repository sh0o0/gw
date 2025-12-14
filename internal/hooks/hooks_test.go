package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHook_shouldExecuteCommands_whenConfigSet(t *testing.T) {
	tDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	outFile := filepath.Join(tDir, "out.txt")
	hookCmd := "echo hello > " + outFile

	cmd = exec.Command("git", "config", "--local", "gw.hooks.postCheckout", hookCmd)
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}

	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": "main",
		"GW_NEW_BRANCH":  "feature",
	}

	ran, err := RunHook(tDir, "post-checkout", env)
	if !ran || err != nil {
		t.Fatalf("hook did not run successfully: ran=%v err=%v", ran, err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read outfile: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "hello" {
		t.Fatalf("expected 'hello', got '%s'", got)
	}
}

func TestRunHook_shouldReturnFalse_whenNoConfig(t *testing.T) {
	tDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	ran, err := RunHook(tDir, "post-checkout", nil)
	if ran {
		t.Fatalf("expected hook not to run when no config set")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunHook_shouldExecuteMultipleCommands_whenMultipleConfigValues(t *testing.T) {
	tDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	outFile := filepath.Join(tDir, "out.txt")

	cmd = exec.Command("git", "config", "--local", "--add", "gw.hooks.postCheckout", "echo first >> "+outFile)
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config add 1: %v", err)
	}

	cmd = exec.Command("git", "config", "--local", "--add", "gw.hooks.postCheckout", "echo second >> "+outFile)
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config add 2: %v", err)
	}

	ran, err := RunHook(tDir, "post-checkout", nil)
	if !ran || err != nil {
		t.Fatalf("hook did not run successfully: ran=%v err=%v", ran, err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read outfile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || lines[0] != "first" || lines[1] != "second" {
		t.Fatalf("expected 'first\\nsecond', got '%s'", string(data))
	}
}

func TestRunHook_shouldPassEnvVariables(t *testing.T) {
	tDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	outFile := filepath.Join(tDir, "out.txt")
	hookCmd := "echo $GW_NEW_BRANCH > " + outFile

	cmd = exec.Command("git", "config", "--local", "gw.hooks.postCheckout", hookCmd)
	cmd.Dir = tDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}

	env := map[string]string{
		"GW_NEW_BRANCH": "my-feature",
	}

	ran, err := RunHook(tDir, "post-checkout", env)
	if !ran || err != nil {
		t.Fatalf("hook did not run successfully: ran=%v err=%v", ran, err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read outfile: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "my-feature" {
		t.Fatalf("expected 'my-feature', got '%s'", got)
	}
}
