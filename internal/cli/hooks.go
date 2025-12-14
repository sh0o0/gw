package cli

import (
	"fmt"
	"os"

	"github.com/sh0o0/gw/internal/hooks"
)

func runPostCheckout(prevRev, newRev, prevBranch, newBranch string, background bool) {
	cwd, _ := os.Getwd()
	runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, cwd, background)
}

func runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, worktreePath string, background bool) {
	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": prevBranch,
		"GW_NEW_BRANCH":  newBranch,
	}
	opts := hooks.Options{Background: background}
	ran, err := hooks.RunHook(worktreePath, "post-checkout", env, opts)
	if !ran {
		return
	}
	if background {
		fmt.Fprintf(os.Stderr, "post-checkout hook started (background), log: %s\n", hooks.LogFile(worktreePath, "post-checkout"))
	} else {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: post-checkout hook failed: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "post-checkout hook executed")
		}
	}
}

func runPostRemove(branch, worktreePath string, background bool) {
	env := map[string]string{
		"GW_HOOK_NAME":      "post-remove",
		"GW_REMOVED_BRANCH": branch,
		"GW_REMOVED_PATH":   worktreePath,
	}
	cwd, _ := os.Getwd()
	opts := hooks.Options{Background: background}
	ran, err := hooks.RunHook(cwd, "post-remove", env, opts)
	if !ran {
		return
	}
	if background {
		fmt.Fprintf(os.Stderr, "post-remove hook started (background), log: %s\n", hooks.LogFile(cwd, "post-remove"))
	} else {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: post-remove hook failed: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "post-remove hook executed")
		}
	}
}
