package cli

import (
	"fmt"
	"os"

	"github.com/sh0o0/gw/internal/hooks"
)

func runPostCheckout(prevRev, newRev, prevBranch, newBranch string) {
	cwd, _ := os.Getwd()
	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": prevBranch,
		"GW_NEW_BRANCH":  newBranch,
	}
	if ran, err := hooks.RunHook(cwd, "post-checkout", env); ran {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: post-checkout hook completed with errors")
		} else {
			fmt.Fprintln(os.Stderr, "post-checkout hook executed")
		}
	}
}

func runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, worktreePath string) {
	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": prevBranch,
		"GW_NEW_BRANCH":  newBranch,
	}
	if ran, err := hooks.RunHook(worktreePath, "post-checkout", env); ran {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: post-checkout hook completed with errors")
		} else {
			fmt.Fprintln(os.Stderr, "post-checkout hook executed")
		}
	}
}
