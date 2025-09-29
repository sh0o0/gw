package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
)

func HooksDir(primaryRoot, gitRoot string) string {
	if primaryRoot != "" {
		return filepath.Join(primaryRoot, ".gw", "hooks")
	}
	return filepath.Join(gitRoot, ".gw", "hooks")
}

// RunHook executes a hook file and all executables in hook.d directory.
// Returns true if any hook ran and nil error if all successful.
func RunHook(dir, name string, env map[string]string, args ...string) (ran bool, err error) {
	if dir == "" {
		return false, nil
	}
	if fi, err2 := os.Stat(dir); err2 != nil || !fi.IsDir() {
		return false, nil
	}
	run := func(path string) error {
		cmd := exec.Command(path, args...)
		cmd.Dir = dir
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	hook := filepath.Join(dir, name)
	if fi, err2 := os.Stat(hook); err2 == nil && fi.Mode().Perm()&0o111 != 0 && !fi.IsDir() {
		if err := run(hook); err != nil {
			return true, err
		}
		ran = true
	}
	d := filepath.Join(dir, name+".d")
	if entries, err2 := os.ReadDir(d); err2 == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(d, e.Name())
			if fi, err3 := os.Stat(p); err3 == nil && fi.Mode().Perm()&0o111 != 0 {
				if err := run(p); err != nil {
					return true, err
				}
				ran = true
			}
		}
	}
	return ran, nil
}
