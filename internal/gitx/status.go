package gitx

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type BranchStatus string

const (
	BranchStatusMerged     BranchStatus = "merged"
	BranchStatusClosed     BranchStatus = "closed"
	BranchStatusOpened     BranchStatus = "opened"
	BranchStatusInProgress BranchStatus = "in progress"
	BranchStatusNotStarted BranchStatus = "not started"
)

type PRInfo struct {
	Status    BranchStatus
	Assignees []string
}

type BranchStatusResolver struct {
	baseRef string
	ghPath  string
	prCache map[string]PRInfo
	mu      sync.Mutex
}

func NewBranchStatusResolver(cwd string) *BranchStatusResolver {
	base := detectBaseRef(cwd)
	ghPath, _ := exec.LookPath("gh")
	return &BranchStatusResolver{
		baseRef: base,
		ghPath:  ghPath,
		prCache: make(map[string]PRInfo),
	}
}

func (r *BranchStatusResolver) Status(path, branch string) (BranchStatus, error) {
	info := r.StatusInfo(path, branch)
	return info.Status, nil
}

func (r *BranchStatusResolver) StatusInfo(path, branch string) PRInfo {
	if branch == "" || branch == "HEAD" {
		return PRInfo{}
	}
	if info := r.prInfo(path, branch); info.Status != "" {
		return info
	}
	changed, err := hasWorkingChanges(path)
	if err != nil {
		return PRInfo{}
	}
	if changed {
		return PRInfo{Status: BranchStatusInProgress}
	}
	ahead, err := r.hasLocalCommits(path)
	if err != nil {
		return PRInfo{}
	}
	if ahead {
		return PRInfo{Status: BranchStatusInProgress}
	}
	return PRInfo{Status: BranchStatusNotStarted}
}

type ghPRResponse struct {
	State     string `json:"state"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
}

func (r *BranchStatusResolver) prInfo(path, branch string) PRInfo {
	if r.ghPath == "" {
		return PRInfo{}
	}
	r.mu.Lock()
	if info, ok := r.prCache[branch]; ok {
		r.mu.Unlock()
		return info
	}
	r.mu.Unlock()
	cmd := exec.Command(r.ghPath, "pr", "view", branch, "--json", "state,assignees")
	cmd.Dir = path
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		r.mu.Lock()
		r.prCache[branch] = PRInfo{}
		r.mu.Unlock()
		return PRInfo{}
	}
	var resp ghPRResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		r.mu.Lock()
		r.prCache[branch] = PRInfo{}
		r.mu.Unlock()
		return PRInfo{}
	}
	info := PRInfo{
		Status: branchStatusFromPRState(strings.TrimSpace(resp.State)),
	}
	for _, a := range resp.Assignees {
		if a.Login != "" {
			info.Assignees = append(info.Assignees, a.Login)
		}
	}
	r.mu.Lock()
	r.prCache[branch] = info
	r.mu.Unlock()
	return info
}

func branchStatusFromPRState(state string) BranchStatus {
	switch strings.ToUpper(state) {
	case "MERGED":
		return BranchStatusMerged
	case "CLOSED":
		return BranchStatusClosed
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

// isMergedIntoBase returns true if all commits on branch are already contained in baseRef.
// It checks rev-list count of commits reachable from branch but not from baseRef.
// Note: We intentionally do not infer MERGED purely from Git ancestry here.
// Use gh PR status to mark MERGED; otherwise fall back to in-progress/not-started.

func (s BranchStatus) String() string {
	return string(s)
}

func (s BranchStatus) Display() string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s))
}
