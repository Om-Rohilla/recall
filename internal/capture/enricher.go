package capture

import (
	"os"

	appctx "github.com/Om-Rohilla/recall/internal/context"
)

type EnrichmentData struct {
	Cwd         string
	GitRepo     string
	GitBranch   string
	ProjectType string
}

// Enrich gathers context signals from the current environment.
// Delegates to the canonical context detection in internal/context/.
func Enrich(cwd string) EnrichmentData {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	ctx := appctx.Detect(cwd)

	return EnrichmentData{
		Cwd:         cwd,
		GitRepo:     ctx.Git.RepoName,
		GitBranch:   ctx.Git.Branch,
		ProjectType: ctx.Project.Type,
	}
}
