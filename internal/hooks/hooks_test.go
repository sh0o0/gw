package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

// Test that RunHook runs with GW_HOOK_CWD as working directory when provided.
func TestRunHook_shouldUseProvidedCWD_whenGW_HOOK_CWDIsSet(t *testing.T) {
	tDir := t.TempDir()
	hooksDir := filepath.Join(tDir, ".gw", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	targetDir := filepath.Join(tDir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	// Create a simple shell hook that writes PWD to a file specified by OUTFILE
	hookPath := filepath.Join(hooksDir, "post-checkout")
	script := "#!/bin/sh\n: > \"$OUTFILE\"\npwd > \"$OUTFILE\"\n"
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	outFile := filepath.Join(targetDir, "pwd.txt")
	env := map[string]string{
		"GW_HOOK_CWD": targetDir,
		"OUTFILE":     outFile,
	}

	ran, err := RunHook(hooksDir, "post-checkout", env, "prev", "new", "1")
	if !ran || err != nil {
		t.Fatalf("hook did not run successfully: ran=%v err=%v", ran, err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read outfile: %v", err)
	}
	got := string(bytesTrimSpace(data))
	// Normalize for macOS where /var may resolve to /private/var
	gotReal, _ := filepath.EvalSymlinks(got)
	targetReal, _ := filepath.EvalSymlinks(targetDir)
	if gotReal != targetReal {
		t.Fatalf("expected hook PWD=%s, got %s", targetReal, gotReal)
	}
}

// bytesTrimSpace is a tiny helper to avoid importing strings package for a single call
func bytesTrimSpace(b []byte) string {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return string(b[i:j])
}
