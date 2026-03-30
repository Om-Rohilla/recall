package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitState struct {
	IsRepo   bool   `json:"is_repo"`
	RepoName string `json:"repo_name,omitempty"`
	Branch   string `json:"branch,omitempty"`
	IsDirty  bool   `json:"is_dirty"`
	RepoRoot string `json:"repo_root,omitempty"`
}

func DetectGit(cwd string) GitState {
	if cwd == "" {
		return GitState{}
	}

	repoRoot := findGitRoot(cwd)
	if repoRoot == "" {
		return GitState{}
	}

	state := GitState{
		IsRepo:   true,
		RepoName: filepath.Base(repoRoot),
		RepoRoot: repoRoot,
	}

	state.Branch = gitBranch(repoRoot)
	state.IsDirty = gitIsDirty(repoRoot)

	return state
}

func findGitRoot(cwd string) string {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if info.IsDir() {
				return dir
			}
			if !info.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func gitBranch(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitIsDirty(repoRoot string) bool {
	out, err := exec.Command("git", "-C", repoRoot, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
