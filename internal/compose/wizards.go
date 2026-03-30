package compose

import (
	"fmt"
	"strings"
)

func (p *Prompter) ComposeFindCommand() string {
	var parts []string
	parts = append(parts, "find")

	dir := p.PromptInput("Search directory?", ".")
	parts = append(parts, dir)

	fileType := p.PromptChoice("What to find?", []string{"files", "directories", "both"}, "files")
	switch fileType {
	case "files":
		parts = append(parts, "-type f")
	case "directories":
		parts = append(parts, "-type d")
	}

	namePattern := p.PromptInput("Name pattern? (e.g., *.log — leave empty to skip)", "")
	if namePattern != "" {
		parts = append(parts, fmt.Sprintf("-name '%s'", ShellEscape(namePattern)))
	}

	sizeFilter := p.PromptInput("Size filter? (e.g., +100M, -1K — leave empty to skip)", "")
	if sizeFilter != "" {
		parts = append(parts, "-size", sizeFilter)
	}

	timeFilter := p.PromptInput("Modified time? (e.g., +30 = older than 30 days, -7 = newer — leave empty to skip)", "")
	if timeFilter != "" {
		parts = append(parts, "-mtime", timeFilter)
	}

	action := p.PromptChoice("What to do with results?", []string{"list", "delete", "print paths", "count", "exec command"}, "list")
	switch action {
	case "list":
		parts = append(parts, "-exec ls -lh {} \\;")
	case "delete":
		parts = append(parts, "-delete")
	case "print paths":
		parts = append(parts, "-print")
	case "count":
		return strings.Join(parts, " ") + " -print | wc -l"
	case "exec command":
		execCmd := p.PromptInput("Command to execute ({} is replaced with filename):", "ls -lh {}")
		parts = append(parts, fmt.Sprintf("-exec %s \\;", execCmd))
	}

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeTarCommand() string {
	action := p.PromptChoice("What do you want to do?", []string{"create archive", "extract archive", "list contents"}, "create archive")

	var parts []string
	parts = append(parts, "tar")

	switch action {
	case "create archive":
		compress := p.PromptChoice("Compression?", []string{"gzip (.tar.gz)", "bzip2 (.tar.bz2)", "xz (.tar.xz)", "none (.tar)"}, "gzip (.tar.gz)")
		switch compress {
		case "gzip (.tar.gz)":
			parts = append(parts, "-czf")
		case "bzip2 (.tar.bz2)":
			parts = append(parts, "-cjf")
		case "xz (.tar.xz)":
			parts = append(parts, "-cJf")
		default:
			parts = append(parts, "-cf")
		}
		output := p.PromptInput("Archive name:", "archive.tar.gz")
		parts = append(parts, output)
		source := p.PromptInput("Source directory/files:", ".")
		parts = append(parts, source)

		exclude := p.PromptInput("Exclude pattern? (leave empty to skip)", "")
		if exclude != "" {
			parts = append(parts, fmt.Sprintf("--exclude='%s'", ShellEscape(exclude)))
		}

	case "extract archive":
		parts = append(parts, "-xf")
		archive := p.PromptInput("Archive file:", "")
		parts = append(parts, archive)

		destDir := p.PromptInput("Extract to directory? (leave empty for current)", "")
		if destDir != "" {
			parts = append(parts, "-C", destDir)
		}

	case "list contents":
		parts = append(parts, "-tf")
		archive := p.PromptInput("Archive file:", "")
		parts = append(parts, archive)
	}

	verbose := p.PromptYesNo("Verbose output?", true)
	if verbose {
		if len(parts) > 1 {
			flags := parts[1]
			if strings.HasPrefix(flags, "-") && !strings.Contains(flags, "v") {
				parts[1] = flags[:2] + "v" + flags[2:]
			}
		}
	}

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeGrepCommand() string {
	var parts []string
	parts = append(parts, "grep")

	var flags []string

	recursive := p.PromptYesNo("Search recursively?", true)
	if recursive {
		flags = append(flags, "-r")
	}

	lineNumbers := p.PromptYesNo("Show line numbers?", true)
	if lineNumbers {
		flags = append(flags, "-n")
	}

	caseInsensitive := p.PromptYesNo("Case-insensitive?", false)
	if caseInsensitive {
		flags = append(flags, "-i")
	}

	if len(flags) > 0 {
		parts = append(parts, strings.Join(flags, ""))
	}

	pattern := p.PromptInput("Search pattern:", "")
	if strings.Contains(pattern, " ") || strings.Contains(pattern, "*") {
		parts = append(parts, fmt.Sprintf("'%s'", ShellEscape(pattern)))
	} else {
		parts = append(parts, pattern)
	}

	includeFilter := p.PromptInput("File type filter? (e.g., *.go, *.py — leave empty for all)", "")
	if includeFilter != "" {
		parts = append(parts, fmt.Sprintf("--include='%s'", ShellEscape(includeFilter)))
	}

	searchDir := p.PromptInput("Directory to search:", ".")
	parts = append(parts, searchDir)

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeDockerCommand() string {
	action := p.PromptChoice("Docker action?", []string{
		"run container", "build image", "list containers", "list images",
		"stop container", "remove containers", "compose up", "compose down",
		"view logs", "exec in container", "system cleanup",
	}, "run container")

	var cmd string
	switch action {
	case "run container":
		image := p.PromptInput("Image name:", "")
		detach := p.PromptYesNo("Run in background?", true)
		name := p.PromptInput("Container name? (leave empty to skip)", "")
		ports := p.PromptInput("Port mapping? (e.g., 8080:80 — leave empty to skip)", "")
		var parts []string
		parts = append(parts, "docker run")
		if detach {
			parts = append(parts, "-d")
		}
		if name != "" {
			parts = append(parts, "--name", name)
		}
		if ports != "" {
			parts = append(parts, "-p", ports)
		}
		parts = append(parts, image)
		cmd = strings.Join(parts, " ")

	case "build image":
		tag := p.PromptInput("Image tag (name:version):", "myapp:latest")
		cmd = fmt.Sprintf("docker build -t %s .", tag)

	case "list containers":
		showAll := p.PromptYesNo("Include stopped containers?", false)
		if showAll {
			cmd = "docker ps -a"
		} else {
			cmd = "docker ps"
		}

	case "list images":
		cmd = "docker images"

	case "stop container":
		container := p.PromptInput("Container name/ID:", "")
		cmd = "docker stop " + container

	case "remove containers":
		allStopped := p.PromptYesNo("Remove ALL stopped containers?", false)
		if allStopped {
			cmd = "docker container prune -f"
		} else {
			container := p.PromptInput("Container name/ID:", "")
			cmd = "docker rm " + container
		}

	case "compose up":
		detach := p.PromptYesNo("Run in background?", true)
		if detach {
			cmd = "docker compose up -d"
		} else {
			cmd = "docker compose up"
		}

	case "compose down":
		removeVolumes := p.PromptYesNo("Also remove volumes?", false)
		if removeVolumes {
			cmd = "docker compose down -v"
		} else {
			cmd = "docker compose down"
		}

	case "view logs":
		container := p.PromptInput("Container name/ID:", "")
		follow := p.PromptYesNo("Follow (live) logs?", true)
		tail := p.PromptInput("Number of lines? (leave empty for all)", "100")
		parts := []string{"docker logs"}
		if follow {
			parts = append(parts, "-f")
		}
		if tail != "" {
			parts = append(parts, "--tail", tail)
		}
		parts = append(parts, container)
		cmd = strings.Join(parts, " ")

	case "exec in container":
		container := p.PromptInput("Container name/ID:", "")
		shell := p.PromptChoice("Shell?", []string{"bash", "sh", "custom"}, "bash")
		if shell == "custom" {
			shell = p.PromptInput("Command to run:", "")
		}
		cmd = fmt.Sprintf("docker exec -it %s %s", container, shell)

	case "system cleanup":
		cmd = "docker system prune -af --volumes"
	}

	return cmd
}

func (p *Prompter) ComposeGitCommand() string {
	action := p.PromptChoice("Git action?", []string{
		"commit", "push", "pull", "branch", "checkout",
		"merge", "stash", "log", "diff", "reset",
	}, "commit")

	var cmd string
	switch action {
	case "commit":
		msg := ShellEscape(p.PromptInput("Commit message:", ""))
		stageAll := p.PromptYesNo("Stage all changes?", true)
		if stageAll {
			cmd = fmt.Sprintf("git add -A && git commit -m '%s'", msg)
		} else {
			cmd = fmt.Sprintf("git commit -m '%s'", msg)
		}

	case "push":
		branch := p.PromptInput("Branch? (leave empty for current)", "")
		if branch != "" {
			cmd = "git push origin " + branch
		} else {
			cmd = "git push"
		}

	case "pull":
		rebase := p.PromptYesNo("Rebase instead of merge?", false)
		if rebase {
			cmd = "git pull --rebase"
		} else {
			cmd = "git pull"
		}

	case "branch":
		subAction := p.PromptChoice("What to do?", []string{"create new", "list all", "delete"}, "create new")
		switch subAction {
		case "create new":
			name := p.PromptInput("Branch name:", "")
			cmd = "git checkout -b " + name
		case "list all":
			cmd = "git branch -a"
		case "delete":
			name := p.PromptInput("Branch to delete:", "")
			cmd = "git branch -d " + name
		}

	case "checkout":
		target := p.PromptInput("Branch or commit:", "")
		cmd = "git checkout " + target

	case "merge":
		branch := p.PromptInput("Branch to merge:", "")
		noFF := p.PromptYesNo("Create merge commit (--no-ff)?", false)
		if noFF {
			cmd = "git merge --no-ff " + branch
		} else {
			cmd = "git merge " + branch
		}

	case "stash":
		subAction := p.PromptChoice("Stash action?", []string{"save", "pop", "list", "drop"}, "save")
		switch subAction {
		case "save":
			msg := p.PromptInput("Stash message? (leave empty for default)", "")
			if msg != "" {
				cmd = fmt.Sprintf("git stash push -m '%s'", ShellEscape(msg))
			} else {
				cmd = "git stash"
			}
		case "pop":
			cmd = "git stash pop"
		case "list":
			cmd = "git stash list"
		case "drop":
			cmd = "git stash drop"
		}

	case "log":
		style := p.PromptChoice("Log format?", []string{"oneline", "graph", "detailed"}, "oneline")
		switch style {
		case "oneline":
			cmd = "git log --oneline"
		case "graph":
			cmd = "git log --oneline --graph --all"
		case "detailed":
			cmd = "git log --stat"
		}
		count := p.PromptInput("Number of commits? (leave empty for all)", "10")
		if count != "" {
			cmd += " -" + count
		}

	case "diff":
		what := p.PromptChoice("Diff what?", []string{"unstaged", "staged", "between branches"}, "unstaged")
		switch what {
		case "unstaged":
			cmd = "git diff"
		case "staged":
			cmd = "git diff --cached"
		case "between branches":
			branch := p.PromptInput("Compare with branch:", "main")
			cmd = "git diff " + branch
		}

	case "reset":
		resetType := p.PromptChoice("Reset type?", []string{"soft (keep staged)", "mixed (keep working tree)", "hard (discard all)"}, "soft (keep staged)")
		ref := p.PromptInput("Reset to?", "HEAD~1")
		switch resetType {
		case "soft (keep staged)":
			cmd = "git reset --soft " + ref
		case "mixed (keep working tree)":
			cmd = "git reset --mixed " + ref
		case "hard (discard all)":
			cmd = "git reset --hard " + ref
		}
	}

	return cmd
}

func (p *Prompter) ComposeCurlCommand() string {
	url := p.PromptInput("URL:", "")
	method := p.PromptChoice("HTTP method?", []string{"GET", "POST", "PUT", "DELETE", "HEAD"}, "GET")

	parts := []string{"curl"}

	if method != "GET" {
		parts = append(parts, "-X", method)
	}

	if method == "POST" || method == "PUT" {
		contentType := p.PromptChoice("Content type?", []string{"json", "form", "none"}, "json")
		switch contentType {
		case "json":
			parts = append(parts, "-H", "'Content-Type: application/json'")
			data := p.PromptInput("JSON data:", "{}")
			parts = append(parts, "-d", fmt.Sprintf("'%s'", ShellEscape(data)))
		case "form":
			data := p.PromptInput("Form data (key=value):", "")
			parts = append(parts, "-d", fmt.Sprintf("'%s'", ShellEscape(data)))
		}
	}

	if method == "HEAD" {
		parts = append(parts, "-I")
	}

	headers := p.PromptInput("Custom header? (e.g., Authorization: Bearer token — leave empty to skip)", "")
	if headers != "" {
		parts = append(parts, "-H", fmt.Sprintf("'%s'", ShellEscape(headers)))
	}

	verbose := p.PromptYesNo("Verbose output?", false)
	if verbose {
		parts = append(parts, "-v")
	}

	silent := p.PromptYesNo("Silent mode (hide progress)?", true)
	if silent {
		parts = append(parts, "-s")
	}

	followRedirects := p.PromptYesNo("Follow redirects?", true)
	if followRedirects {
		parts = append(parts, "-L")
	}

	parts = append(parts, url)

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeSSHCommand() string {
	target := p.PromptInput("Target (user@host):", "")
	parts := []string{"ssh"}

	port := p.PromptInput("Port? (leave empty for 22)", "")
	if port != "" && port != "22" {
		parts = append(parts, "-p", port)
	}

	keyFile := p.PromptInput("Identity file? (leave empty for default)", "")
	if keyFile != "" {
		parts = append(parts, "-i", keyFile)
	}

	tunnelType := p.PromptChoice("Port forwarding?", []string{"none", "local", "remote", "dynamic SOCKS"}, "none")
	switch tunnelType {
	case "local":
		localPort := p.PromptInput("Local port:", "8080")
		remoteHost := p.PromptInput("Remote host:port:", "localhost:80")
		parts = append(parts, "-L", fmt.Sprintf("%s:%s", localPort, remoteHost))
	case "remote":
		remotePort := p.PromptInput("Remote port:", "8080")
		localHost := p.PromptInput("Local host:port:", "localhost:80")
		parts = append(parts, "-R", fmt.Sprintf("%s:%s", remotePort, localHost))
	case "dynamic SOCKS":
		socksPort := p.PromptInput("SOCKS port:", "1080")
		parts = append(parts, "-D", socksPort)
	}

	parts = append(parts, target)

	remoteCmd := p.PromptInput("Remote command? (leave empty for shell)", "")
	if remoteCmd != "" {
		parts = append(parts, fmt.Sprintf("'%s'", ShellEscape(remoteCmd)))
	}

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeRsyncCommand() string {
	source := p.PromptInput("Source path:", "")
	dest := p.PromptInput("Destination path:", "")

	parts := []string{"rsync", "-av"}

	compress := p.PromptYesNo("Compress during transfer?", true)
	if compress {
		parts[1] += "z"
	}

	showProgress := p.PromptYesNo("Show progress?", true)
	if showProgress {
		parts = append(parts, "--progress")
	}

	deleteExtra := p.PromptYesNo("Delete files in destination not in source?", false)
	if deleteExtra {
		parts = append(parts, "--delete")
	}

	exclude := p.PromptInput("Exclude pattern? (leave empty to skip)", "")
	if exclude != "" {
		parts = append(parts, fmt.Sprintf("--exclude='%s'", ShellEscape(exclude)))
	}

	dryRun := p.PromptYesNo("Dry run first?", false)
	if dryRun {
		parts = append(parts, "--dry-run")
	}

	parts = append(parts, source, dest)

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeChmodCommand() string {
	modeType := p.PromptChoice("Permission style?", []string{"numeric (e.g., 755)", "symbolic (e.g., u+x)"}, "numeric (e.g., 755)")

	var mode string
	if modeType == "numeric (e.g., 755)" {
		mode = p.PromptInput("Permission mode (e.g., 755, 644, 700):", "755")
	} else {
		mode = p.PromptInput("Symbolic mode (e.g., u+x, g-w, o=r):", "u+x")
	}

	target := p.PromptInput("File or directory:", "")
	recursive := p.PromptYesNo("Apply recursively?", false)

	parts := []string{"chmod"}
	if recursive {
		parts = append(parts, "-R")
	}
	parts = append(parts, mode, target)

	return strings.Join(parts, " ")
}

func (p *Prompter) ComposeGeneric(tool string) string {
	fmt.Println(p.styles.Metadata("  (No wizard template for this tool — entering free-form mode)"))
	fmt.Println()

	var parts []string
	parts = append(parts, tool)

	for {
		arg := p.PromptInput("Add flag or argument (leave empty when done):", "")
		if arg == "" {
			break
		}
		parts = append(parts, arg)
	}

	return strings.Join(parts, " ")
}

// SupportedTools returns the list of tools with dedicated wizards.
var SupportedTools = map[string]bool{
	"find": true, "tar": true, "grep": true, "docker": true,
	"git": true, "curl": true, "ssh": true, "rsync": true, "chmod": true,
}

// Route dispatches to the appropriate wizard for a given tool.
func (p *Prompter) Route(tool string) string {
	switch tool {
	case "find":
		return p.ComposeFindCommand()
	case "tar":
		return p.ComposeTarCommand()
	case "grep":
		return p.ComposeGrepCommand()
	case "docker":
		return p.ComposeDockerCommand()
	case "git":
		return p.ComposeGitCommand()
	case "curl":
		return p.ComposeCurlCommand()
	case "ssh":
		return p.ComposeSSHCommand()
	case "rsync":
		return p.ComposeRsyncCommand()
	case "chmod":
		return p.ComposeChmodCommand()
	default:
		return p.ComposeGeneric(tool)
	}
}
