package capture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type EnrichmentData struct {
	Cwd         string
	GitRepo     string
	GitBranch   string
	ProjectType string
}

// Enrich gathers context signals from the current environment.
func Enrich(cwd string) EnrichmentData {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	data := EnrichmentData{
		Cwd: cwd,
	}

	data.GitRepo, data.GitBranch = detectGit(cwd)
	data.ProjectType = detectProjectType(cwd)

	return data
}

func detectGit(cwd string) (repo, branch string) {
	// Walk up to find .git directory
	dir := cwd
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			repo = filepath.Base(dir)
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}

	// Get current branch
	out, err := exec.Command("git", "-C", cwd, "branch", "--show-current").Output()
	if err == nil {
		branch = strings.TrimSpace(string(out))
	}

	return repo, branch
}

func detectProjectType(cwd string) string {
	projectFiles := map[string]string{
		"package.json":    "node",
		"go.mod":          "go",
		"Cargo.toml":      "rust",
		"requirements.txt": "python",
		"setup.py":        "python",
		"pyproject.toml":  "python",
		"Pipfile":         "python",
		"Gemfile":         "ruby",
		"pom.xml":         "java",
		"build.gradle":    "java",
		"composer.json":   "php",
		"Makefile":        "make",
		"CMakeLists.txt":  "cmake",
		"Dockerfile":      "docker",
		"docker-compose.yml":  "docker",
		"docker-compose.yaml": "docker",
		".terraform":      "terraform",
		"main.tf":         "terraform",
	}

	// Check current directory
	for file, ptype := range projectFiles {
		if _, err := os.Stat(filepath.Join(cwd, file)); err == nil {
			return ptype
		}
	}

	// Walk up (max 3 levels) to find project root
	dir := cwd
	for i := 0; i < 3; i++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		for file, ptype := range projectFiles {
			if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
				return ptype
			}
		}
	}

	return ""
}
