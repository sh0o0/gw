package cli

import (
	"fmt"
	"os"

	"github.com/sh0o0/gw/internal/hooks"
)

func runPostCreate(branch, worktreePath string, background bool) {
	env := map[string]string{
		"GW_HOOK_NAME": "post-create",
		"GW_BRANCH":    branch,
		"GW_PATH":      worktreePath,
	}
	opts := hooks.Options{Background: background}
	ran, err := hooks.RunHook(worktreePath, "post-create", env, opts)
	if !ran {
		return
	}
	if background {
		fmt.Fprintf(os.Stderr, "post-create hook started (background), log: %s\n", hooks.LogFile(worktreePath, "post-create"))
	} else {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: post-create hook failed: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "post-create hook executed")
		}
	}
}
