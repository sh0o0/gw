package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sh0o0/gw/internal/gitx"
)

type Options struct {
	Background bool
}

func configKeyForHook(name string) string {
	switch name {
	case "post-checkout":
		return "gw.hooks.postCheckout"
	default:
		return "gw.hooks." + name
	}
}

func logFilePath(worktreePath, name string) string {
	return filepath.Join(worktreePath, "gw-hook-"+name+".log")
}

// LogFile returns the path to the log file for the given hook.
func LogFile(worktreePath, name string) string {
	return logFilePath(worktreePath, name)
}

func buildEnvString(env map[string]string) string {
	var parts []string
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("%s=%q", k, v))
	}
	return strings.Join(parts, " ")
}

// RunHook executes hook commands from git config.
// Commands are read from gw.hook.<name> (multi-value) and executed via sh -c.
// Output is written to <worktreePath>/gw-hook-<name>.log.
// If opts.Background is true, hooks run in a detached process.
// Returns true if any hook ran/started and error (only for foreground execution).
func RunHook(worktreePath, name string, env map[string]string, opts Options) (ran bool, err error) {
	key := configKeyForHook(name)
	cmds, _ := gitx.ConfigGetAll(worktreePath, key)
	if len(cmds) == 0 {
		return false, nil
	}

	logPath := logFilePath(worktreePath, name)

	if opts.Background {
		envStr := buildEnvString(env)
		for _, cmdStr := range cmds {
			if cmdStr == "" {
				continue
			}
			wrappedCmd := fmt.Sprintf(
				"{ echo ''; echo '=== '%s': '%s' ==='; %s %s; } >> %q 2>&1",
				time.Now().Format(time.RFC3339),
				cmdStr,
				envStr,
				cmdStr,
				logPath,
			)
			cmd := exec.Command("sh", "-c", wrappedCmd)
			cmd.Dir = worktreePath
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Start(); err != nil {
				continue
			}
			ran = true
		}
		return ran, nil
	}

	envSlice := os.Environ()
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer logFile.Close()

	for _, cmdStr := range cmds {
		if cmdStr == "" {
			continue
		}
		fmt.Fprintf(logFile, "\n=== %s: %s ===\n", time.Now().Format(time.RFC3339), cmdStr)
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = worktreePath
		cmd.Env = envSlice
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Stdin = nil
		if err := cmd.Run(); err != nil {
			return true, err
		}
		ran = true
	}
	return ran, nil
}
