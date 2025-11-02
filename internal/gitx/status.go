package gitx

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type BranchStatus string

const (
	BranchStatusMerged     BranchStatus = "merged"
	BranchStatusOpened     BranchStatus = "opened"
	BranchStatusInProgress BranchStatus = "in progress"
	BranchStatusNotStarted BranchStatus = "not started"
)

type BranchStatusResolver struct {
	baseRef string
	ghPath  string
	prCache map[string]BranchStatus
	mu      sync.Mutex
}

func NewBranchStatusResolver(cwd string) *BranchStatusResolver {
	base := detectBaseRef(cwd)
	ghPath, _ := exec.LookPath("gh")
	return &BranchStatusResolver{
		baseRef: base,
		ghPath:  ghPath,
		prCache: make(map[string]BranchStatus),
	}
}

func (r *BranchStatusResolver) Status(path, branch string) (BranchStatus, error) {
	if branch == "" || branch == "HEAD" {
		return "", nil
	}
	if st := r.prStatus(path, branch); st != "" {
		return st, nil
	}
	changed, err := hasWorkingChanges(path)
	if err != nil {
		return "", err
	}
	if changed {
		return BranchStatusInProgress, nil
	}
	ahead, err := r.hasLocalCommits(path)
	if err != nil {
		return "", err
	}
	if ahead {
		return BranchStatusInProgress, nil
	}
	return BranchStatusNotStarted, nil
}

func (r *BranchStatusResolver) prStatus(path, branch string) BranchStatus {
	if r.ghPath == "" {
		return ""
	}
	r.mu.Lock()
	if st, ok := r.prCache[branch]; ok {
		r.mu.Unlock()
		return st
	}
	r.mu.Unlock()
	cmd := exec.Command(r.ghPath, "pr", "view", branch, "--json", "state", "--jq", ".state")
	cmd.Dir = path
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		r.mu.Lock()
		r.prCache[branch] = ""
		r.mu.Unlock()
		return ""
	}
	status := branchStatusFromPRState(strings.TrimSpace(out.String()))
	r.mu.Lock()
	r.prCache[branch] = status
	r.mu.Unlock()
	return status
}

func branchStatusFromPRState(state string) BranchStatus {
	switch strings.ToUpper(state) {
	case "MERGED":
		return BranchStatusMerged
	case "OPEN":
		return BranchStatusOpened
	default:
		return ""
	}
}

func (r *BranchStatusResolver) hasLocalCommits(path string) (bool, error) {
	if r.baseRef == "" {
		return false, nil
	}
	if _, err := Cmd(path, "rev-parse", "--verify", r.baseRef); err != nil {
		return false, nil
	}
	out, err := Cmd(path, "rev-list", "--count", r.baseRef+"..HEAD")
	if err != nil {
		return false, nil
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func hasWorkingChanges(path string) (bool, error) {
	out, err := Cmd(path, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func detectBaseRef(cwd string) string {
	if out, err := Cmd(cwd, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		ref := strings.TrimSpace(out)
		if ref != "" {
			return ref
		}
	}
	candidates := []string{"origin/main", "origin/master", "main", "master"}
	for _, candidate := range candidates {
		if _, err := Cmd(cwd, "rev-parse", "--verify", candidate); err == nil {
			return candidate
		}
	}
	if branch, err := BranchAt(cwd); err == nil {
		if branch != "" && branch != "HEAD" {
			return branch
		}
	}
	return ""
}

func (s BranchStatus) String() string {
	return string(s)
}

func (s BranchStatus) Display() string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s))
}
