package hooks

import (
	"os"
	"os/exec"

	"github.com/sh0o0/gw/internal/gitx"
)

func configKeyForHook(name string) string {
	switch name {
	case "post-checkout":
		return "gw.hooks.postCheckout"
	default:
		return "gw.hooks." + name
	}
}

// RunHook executes hook commands from git config.
// Commands are read from gw.hook.<name> (multi-value) and executed via sh -c.
// Returns true if any hook ran and nil error if all successful.
func RunHook(cwd, name string, env map[string]string) (ran bool, err error) {
	key := configKeyForHook(name)
	cmds, _ := gitx.ConfigGetAll(cwd, key)
	if len(cmds) == 0 {
		return false, nil
	}
	for _, cmdStr := range cmds {
		if cmdStr == "" {
			continue
		}
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = cwd
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return true, err
		}
		ran = true
	}
	return ran, nil
}
