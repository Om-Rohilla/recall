package context

import (
	"os"
	"path/filepath"
)

type ProjectInfo struct {
	Type      string `json:"type,omitempty"`
	Framework string `json:"framework,omitempty"`
}

var projectMarkers = []struct {
	File      string
	Type      string
	Framework string
}{
	{"package.json", "node", ""},
	{"go.mod", "go", ""},
	{"Cargo.toml", "rust", ""},
	{"pyproject.toml", "python", ""},
	{"requirements.txt", "python", ""},
	{"setup.py", "python", ""},
	{"Pipfile", "python", ""},
	{"Gemfile", "ruby", ""},
	{"pom.xml", "java", "maven"},
	{"build.gradle", "java", "gradle"},
	{"build.gradle.kts", "kotlin", "gradle"},
	{"composer.json", "php", ""},
	{"mix.exs", "elixir", ""},
	{"shard.yml", "crystal", ""},
	{"pubspec.yaml", "dart", "flutter"},
	{"Makefile", "make", ""},
	{"CMakeLists.txt", "cpp", "cmake"},
	{"Dockerfile", "docker", ""},
	{"docker-compose.yml", "docker", "compose"},
	{"docker-compose.yaml", "docker", "compose"},
	{".terraform", "terraform", ""},
	{"main.tf", "terraform", ""},
	{"serverless.yml", "serverless", ""},
	{"Vagrantfile", "vagrant", ""},
	{"ansible.cfg", "ansible", ""},
	{"Chart.yaml", "helm", ""},
	{"skaffold.yaml", "kubernetes", "skaffold"},
}

var frameworkMarkers = []struct {
	File      string
	Framework string
}{
	{"next.config.js", "nextjs"},
	{"next.config.mjs", "nextjs"},
	{"next.config.ts", "nextjs"},
	{"nuxt.config.js", "nuxt"},
	{"nuxt.config.ts", "nuxt"},
	{"vite.config.js", "vite"},
	{"vite.config.ts", "vite"},
	{"angular.json", "angular"},
	{"svelte.config.js", "svelte"},
	{"remix.config.js", "remix"},
	{"astro.config.mjs", "astro"},
	{"tailwind.config.js", "tailwind"},
	{"tailwind.config.ts", "tailwind"},
	{".flaskenv", "flask"},
	{"manage.py", "django"},
	{"config/routes.rb", "rails"},
}

func DetectProject(cwd string) ProjectInfo {
	if cwd == "" {
		return ProjectInfo{}
	}

	info := ProjectInfo{}

	// Check current directory and walk up (max 4 levels)
	dir := cwd
	for i := 0; i < 5; i++ {
		for _, m := range projectMarkers {
			path := filepath.Join(dir, m.File)
			if _, err := os.Stat(path); err == nil {
				if info.Type == "" {
					info.Type = m.Type
					if m.Framework != "" {
						info.Framework = m.Framework
					}
				}
			}
		}

		if info.Type != "" {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Check for framework-specific markers in the project root
	if info.Type != "" && info.Framework == "" {
		projectRoot := dir
		if projectRoot == cwd {
			projectRoot = cwd
		}
		for _, fm := range frameworkMarkers {
			if _, err := os.Stat(filepath.Join(projectRoot, fm.File)); err == nil {
				info.Framework = fm.Framework
				break
			}
		}
	}

	return info
}
