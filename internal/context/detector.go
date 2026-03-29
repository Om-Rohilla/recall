package context

import "os"

type CurrentContext struct {
	Cwd         string      `json:"cwd"`
	Git         GitState    `json:"git"`
	Project     ProjectInfo `json:"project"`
	SessionID   string      `json:"session_id,omitempty"`
	Environment EnvInfo     `json:"environment"`
}

type EnvInfo struct {
	VirtualEnv string `json:"virtual_env,omitempty"`
	NodeEnv    string `json:"node_env,omitempty"`
	Kubeconfig string `json:"kubeconfig,omitempty"`
	GoPath     string `json:"gopath,omitempty"`
}

func Detect(cwd string) CurrentContext {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	return CurrentContext{
		Cwd:         cwd,
		Git:         DetectGit(cwd),
		Project:     DetectProject(cwd),
		Environment: detectEnv(),
	}
}

func detectEnv() EnvInfo {
	return EnvInfo{
		VirtualEnv: os.Getenv("VIRTUAL_ENV"),
		NodeEnv:    os.Getenv("NODE_ENV"),
		Kubeconfig: os.Getenv("KUBECONFIG"),
		GoPath:     os.Getenv("GOPATH"),
	}
}
