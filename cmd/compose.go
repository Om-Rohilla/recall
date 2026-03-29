package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Om-Rohilla/recall/internal/explain"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/spf13/cobra"
)

var composeCmd = &cobra.Command{
	Use:     "compose",
	Aliases: []string{"c"},
	Short:   "Build a command interactively step-by-step",
	Long: `Compose builds a command through an interactive wizard.
Choose a tool, select options step by step, and get the final command.

Supports: find, tar, grep, docker, git, kubectl, ssh, curl, rsync, chmod, and more.

Example:
  recall compose`,
	RunE: runCompose,
}

func init() {
	rootCmd.AddCommand(composeCmd)
}

func runCompose(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.TitleStyle.Render("Recall Command Composer"))
	fmt.Println()

	// Step 1: Choose tool
	tool := promptInput(reader, "What tool do you want to use?", "")
	if tool == "" {
		return fmt.Errorf("no tool specified")
	}

	toolInfo := explain.GetToolInfo(tool)
	if toolInfo == nil {
		return runGenericCompose(reader, tool)
	}

	fmt.Println(ui.MetadataStyle.Render(fmt.Sprintf("  %s — %s", tool, toolInfo.Description)))
	fmt.Println()

	// Route to tool-specific wizards
	switch tool {
	case "find":
		composeFindCommand(reader)
	case "tar":
		composeTarCommand(reader)
	case "grep":
		composeGrepCommand(reader)
	case "docker":
		composeDockerCommand(reader)
	case "git":
		composeGitCommand(reader)
	case "curl":
		composeCurlCommand(reader)
	case "ssh":
		composeSSHCommand(reader)
	case "rsync":
		composeRsyncCommand(reader)
	case "chmod":
		composeChmodCommand(reader)
	default:
		return runGenericCompose(reader, tool)
	}
	return nil
}

func composeFindCommand(reader *bufio.Reader) string {
	var parts []string
	parts = append(parts, "find")

	dir := promptInput(reader, "Search directory?", ".")
	parts = append(parts, dir)

	fileType := promptChoice(reader, "What to find?", []string{"files", "directories", "both"}, "files")
	switch fileType {
	case "files":
		parts = append(parts, "-type f")
	case "directories":
		parts = append(parts, "-type d")
	}

	namePattern := promptInput(reader, "Name pattern? (e.g., *.log — leave empty to skip)", "")
	if namePattern != "" {
		parts = append(parts, fmt.Sprintf("-name '%s'", namePattern))
	}

	sizeFilter := promptInput(reader, "Size filter? (e.g., +100M, -1K — leave empty to skip)", "")
	if sizeFilter != "" {
		parts = append(parts, "-size", sizeFilter)
	}

	timeFilter := promptInput(reader, "Modified time? (e.g., +30 = older than 30 days, -7 = newer — leave empty to skip)", "")
	if timeFilter != "" {
		parts = append(parts, "-mtime", timeFilter)
	}

	action := promptChoice(reader, "What to do with results?", []string{"list", "delete", "print paths", "count", "exec command"}, "list")
	switch action {
	case "list":
		parts = append(parts, "-exec ls -lh {} \\;")
	case "delete":
		parts = append(parts, "-delete")
	case "print paths":
		parts = append(parts, "-print")
	case "count":
		// Will pipe to wc -l
		parts = append(parts, "-print")
		return finishCompose(reader, strings.Join(parts, " ")+" | wc -l")
	case "exec command":
		execCmd := promptInput(reader, "Command to execute ({} is replaced with filename):", "ls -lh {}")
		parts = append(parts, fmt.Sprintf("-exec %s \\;", execCmd))
	}

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeTarCommand(reader *bufio.Reader) string {
	action := promptChoice(reader, "What do you want to do?", []string{"create archive", "extract archive", "list contents"}, "create archive")

	var parts []string
	parts = append(parts, "tar")

	switch action {
	case "create archive":
		compress := promptChoice(reader, "Compression?", []string{"gzip (.tar.gz)", "bzip2 (.tar.bz2)", "xz (.tar.xz)", "none (.tar)"}, "gzip (.tar.gz)")
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
		output := promptInput(reader, "Archive name:", "archive.tar.gz")
		parts = append(parts, output)
		source := promptInput(reader, "Source directory/files:", ".")
		parts = append(parts, source)

		exclude := promptInput(reader, "Exclude pattern? (leave empty to skip)", "")
		if exclude != "" {
			parts = append(parts, fmt.Sprintf("--exclude='%s'", exclude))
		}

	case "extract archive":
		parts = append(parts, "-xf")
		archive := promptInput(reader, "Archive file:", "")
		parts = append(parts, archive)

		destDir := promptInput(reader, "Extract to directory? (leave empty for current)", "")
		if destDir != "" {
			parts = append(parts, "-C", destDir)
		}

	case "list contents":
		parts = append(parts, "-tf")
		archive := promptInput(reader, "Archive file:", "")
		parts = append(parts, archive)
	}

	verbose := promptYesNo(reader, "Verbose output?", true)
	if verbose {
		// Insert -v into the flags
		if len(parts) > 1 {
			flags := parts[1]
			if strings.HasPrefix(flags, "-") && !strings.Contains(flags, "v") {
				parts[1] = flags[:2] + "v" + flags[2:]
			}
		}
	}

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeGrepCommand(reader *bufio.Reader) string {
	var parts []string
	parts = append(parts, "grep")

	var flags []string

	recursive := promptYesNo(reader, "Search recursively?", true)
	if recursive {
		flags = append(flags, "-r")
	}

	lineNumbers := promptYesNo(reader, "Show line numbers?", true)
	if lineNumbers {
		flags = append(flags, "-n")
	}

	caseInsensitive := promptYesNo(reader, "Case-insensitive?", false)
	if caseInsensitive {
		flags = append(flags, "-i")
	}

	if len(flags) > 0 {
		parts = append(parts, strings.Join(flags, ""))
	}

	pattern := promptInput(reader, "Search pattern:", "")
	if strings.Contains(pattern, " ") || strings.Contains(pattern, "*") {
		parts = append(parts, fmt.Sprintf("'%s'", pattern))
	} else {
		parts = append(parts, pattern)
	}

	includeFilter := promptInput(reader, "File type filter? (e.g., *.go, *.py — leave empty for all)", "")
	if includeFilter != "" {
		parts = append(parts, fmt.Sprintf("--include='%s'", includeFilter))
	}

	searchDir := promptInput(reader, "Directory to search:", ".")
	parts = append(parts, searchDir)

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeDockerCommand(reader *bufio.Reader) string {
	action := promptChoice(reader, "Docker action?", []string{
		"run container", "build image", "list containers", "list images",
		"stop container", "remove containers", "compose up", "compose down",
		"view logs", "exec in container", "system cleanup",
	}, "run container")

	var cmd string
	switch action {
	case "run container":
		image := promptInput(reader, "Image name:", "")
		detach := promptYesNo(reader, "Run in background?", true)
		name := promptInput(reader, "Container name? (leave empty to skip)", "")
		ports := promptInput(reader, "Port mapping? (e.g., 8080:80 — leave empty to skip)", "")
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
		tag := promptInput(reader, "Image tag (name:version):", "myapp:latest")
		cmd = fmt.Sprintf("docker build -t %s .", tag)

	case "list containers":
		showAll := promptYesNo(reader, "Include stopped containers?", false)
		if showAll {
			cmd = "docker ps -a"
		} else {
			cmd = "docker ps"
		}

	case "list images":
		cmd = "docker images"

	case "stop container":
		container := promptInput(reader, "Container name/ID:", "")
		cmd = "docker stop " + container

	case "remove containers":
		allStopped := promptYesNo(reader, "Remove ALL stopped containers?", false)
		if allStopped {
			cmd = "docker container prune -f"
		} else {
			container := promptInput(reader, "Container name/ID:", "")
			cmd = "docker rm " + container
		}

	case "compose up":
		detach := promptYesNo(reader, "Run in background?", true)
		if detach {
			cmd = "docker compose up -d"
		} else {
			cmd = "docker compose up"
		}

	case "compose down":
		removeVolumes := promptYesNo(reader, "Also remove volumes?", false)
		if removeVolumes {
			cmd = "docker compose down -v"
		} else {
			cmd = "docker compose down"
		}

	case "view logs":
		container := promptInput(reader, "Container name/ID:", "")
		follow := promptYesNo(reader, "Follow (live) logs?", true)
		tail := promptInput(reader, "Number of lines? (leave empty for all)", "100")
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
		container := promptInput(reader, "Container name/ID:", "")
		shell := promptChoice(reader, "Shell?", []string{"bash", "sh", "custom"}, "bash")
		if shell == "custom" {
			shell = promptInput(reader, "Command to run:", "")
		}
		cmd = fmt.Sprintf("docker exec -it %s %s", container, shell)

	case "system cleanup":
		cmd = "docker system prune -af --volumes"
	}

	return finishCompose(reader, cmd)
}

func composeGitCommand(reader *bufio.Reader) string {
	action := promptChoice(reader, "Git action?", []string{
		"commit", "push", "pull", "branch", "checkout",
		"merge", "stash", "log", "diff", "reset",
	}, "commit")

	var cmd string
	switch action {
	case "commit":
		msg := promptInput(reader, "Commit message:", "")
		stageAll := promptYesNo(reader, "Stage all changes?", true)
		if stageAll {
			cmd = fmt.Sprintf("git add -A && git commit -m '%s'", msg)
		} else {
			cmd = fmt.Sprintf("git commit -m '%s'", msg)
		}

	case "push":
		branch := promptInput(reader, "Branch? (leave empty for current)", "")
		if branch != "" {
			cmd = "git push origin " + branch
		} else {
			cmd = "git push"
		}

	case "pull":
		rebase := promptYesNo(reader, "Rebase instead of merge?", false)
		if rebase {
			cmd = "git pull --rebase"
		} else {
			cmd = "git pull"
		}

	case "branch":
		subAction := promptChoice(reader, "What to do?", []string{"create new", "list all", "delete"}, "create new")
		switch subAction {
		case "create new":
			name := promptInput(reader, "Branch name:", "")
			cmd = "git checkout -b " + name
		case "list all":
			cmd = "git branch -a"
		case "delete":
			name := promptInput(reader, "Branch to delete:", "")
			cmd = "git branch -d " + name
		}

	case "checkout":
		target := promptInput(reader, "Branch or commit:", "")
		cmd = "git checkout " + target

	case "merge":
		branch := promptInput(reader, "Branch to merge:", "")
		noFF := promptYesNo(reader, "Create merge commit (--no-ff)?", false)
		if noFF {
			cmd = "git merge --no-ff " + branch
		} else {
			cmd = "git merge " + branch
		}

	case "stash":
		subAction := promptChoice(reader, "Stash action?", []string{"save", "pop", "list", "drop"}, "save")
		switch subAction {
		case "save":
			msg := promptInput(reader, "Stash message? (leave empty for default)", "")
			if msg != "" {
				cmd = fmt.Sprintf("git stash push -m '%s'", msg)
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
		style := promptChoice(reader, "Log format?", []string{"oneline", "graph", "detailed"}, "oneline")
		switch style {
		case "oneline":
			cmd = "git log --oneline"
		case "graph":
			cmd = "git log --oneline --graph --all"
		case "detailed":
			cmd = "git log --stat"
		}
		count := promptInput(reader, "Number of commits? (leave empty for all)", "10")
		if count != "" {
			cmd += " -" + count
		}

	case "diff":
		what := promptChoice(reader, "Diff what?", []string{"unstaged", "staged", "between branches"}, "unstaged")
		switch what {
		case "unstaged":
			cmd = "git diff"
		case "staged":
			cmd = "git diff --cached"
		case "between branches":
			branch := promptInput(reader, "Compare with branch:", "main")
			cmd = "git diff " + branch
		}

	case "reset":
		resetType := promptChoice(reader, "Reset type?", []string{"soft (keep staged)", "mixed (keep working tree)", "hard (discard all)"}, "soft (keep staged)")
		ref := promptInput(reader, "Reset to?", "HEAD~1")
		switch resetType {
		case "soft (keep staged)":
			cmd = "git reset --soft " + ref
		case "mixed (keep working tree)":
			cmd = "git reset --mixed " + ref
		case "hard (discard all)":
			cmd = "git reset --hard " + ref
		}
	}

	return finishCompose(reader, cmd)
}

func composeCurlCommand(reader *bufio.Reader) string {
	url := promptInput(reader, "URL:", "")
	method := promptChoice(reader, "HTTP method?", []string{"GET", "POST", "PUT", "DELETE", "HEAD"}, "GET")

	parts := []string{"curl"}

	if method != "GET" {
		parts = append(parts, "-X", method)
	}

	if method == "POST" || method == "PUT" {
		contentType := promptChoice(reader, "Content type?", []string{"json", "form", "none"}, "json")
		switch contentType {
		case "json":
			parts = append(parts, "-H", "'Content-Type: application/json'")
			data := promptInput(reader, "JSON data:", "{}")
			parts = append(parts, "-d", fmt.Sprintf("'%s'", data))
		case "form":
			data := promptInput(reader, "Form data (key=value):", "")
			parts = append(parts, "-d", fmt.Sprintf("'%s'", data))
		}
	}

	if method == "HEAD" {
		parts = append(parts, "-I")
	}

	headers := promptInput(reader, "Custom header? (e.g., Authorization: Bearer token — leave empty to skip)", "")
	if headers != "" {
		parts = append(parts, "-H", fmt.Sprintf("'%s'", headers))
	}

	verbose := promptYesNo(reader, "Verbose output?", false)
	if verbose {
		parts = append(parts, "-v")
	}

	silent := promptYesNo(reader, "Silent mode (hide progress)?", true)
	if silent {
		parts = append(parts, "-s")
	}

	followRedirects := promptYesNo(reader, "Follow redirects?", true)
	if followRedirects {
		parts = append(parts, "-L")
	}

	parts = append(parts, url)

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeSSHCommand(reader *bufio.Reader) string {
	target := promptInput(reader, "Target (user@host):", "")
	parts := []string{"ssh"}

	port := promptInput(reader, "Port? (leave empty for 22)", "")
	if port != "" && port != "22" {
		parts = append(parts, "-p", port)
	}

	keyFile := promptInput(reader, "Identity file? (leave empty for default)", "")
	if keyFile != "" {
		parts = append(parts, "-i", keyFile)
	}

	tunnelType := promptChoice(reader, "Port forwarding?", []string{"none", "local", "remote", "dynamic SOCKS"}, "none")
	switch tunnelType {
	case "local":
		localPort := promptInput(reader, "Local port:", "8080")
		remoteHost := promptInput(reader, "Remote host:port:", "localhost:80")
		parts = append(parts, "-L", fmt.Sprintf("%s:%s", localPort, remoteHost))
	case "remote":
		remotePort := promptInput(reader, "Remote port:", "8080")
		localHost := promptInput(reader, "Local host:port:", "localhost:80")
		parts = append(parts, "-R", fmt.Sprintf("%s:%s", remotePort, localHost))
	case "dynamic SOCKS":
		socksPort := promptInput(reader, "SOCKS port:", "1080")
		parts = append(parts, "-D", socksPort)
	}

	parts = append(parts, target)

	remoteCmd := promptInput(reader, "Remote command? (leave empty for shell)", "")
	if remoteCmd != "" {
		parts = append(parts, fmt.Sprintf("'%s'", remoteCmd))
	}

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeRsyncCommand(reader *bufio.Reader) string {
	source := promptInput(reader, "Source path:", "")
	dest := promptInput(reader, "Destination path:", "")

	parts := []string{"rsync", "-av"}

	compress := promptYesNo(reader, "Compress during transfer?", true)
	if compress {
		parts[1] += "z"
	}

	showProgress := promptYesNo(reader, "Show progress?", true)
	if showProgress {
		parts = append(parts, "--progress")
	}

	deleteExtra := promptYesNo(reader, "Delete files in destination not in source?", false)
	if deleteExtra {
		parts = append(parts, "--delete")
	}

	exclude := promptInput(reader, "Exclude pattern? (leave empty to skip)", "")
	if exclude != "" {
		parts = append(parts, fmt.Sprintf("--exclude='%s'", exclude))
	}

	dryRun := promptYesNo(reader, "Dry run first?", false)
	if dryRun {
		parts = append(parts, "--dry-run")
	}

	parts = append(parts, source, dest)

	return finishCompose(reader, strings.Join(parts, " "))
}

func composeChmodCommand(reader *bufio.Reader) string {
	modeType := promptChoice(reader, "Permission style?", []string{"numeric (e.g., 755)", "symbolic (e.g., u+x)"}, "numeric (e.g., 755)")

	var mode string
	if modeType == "numeric (e.g., 755)" {
		mode = promptInput(reader, "Permission mode (e.g., 755, 644, 700):", "755")
	} else {
		mode = promptInput(reader, "Symbolic mode (e.g., u+x, g-w, o=r):", "u+x")
	}

	target := promptInput(reader, "File or directory:", "")
	recursive := promptYesNo(reader, "Apply recursively?", false)

	parts := []string{"chmod"}
	if recursive {
		parts = append(parts, "-R")
	}
	parts = append(parts, mode, target)

	return finishCompose(reader, strings.Join(parts, " "))
}

func runGenericCompose(reader *bufio.Reader, tool string) error {
	fmt.Println(ui.MetadataStyle.Render("  (No wizard template for this tool — entering free-form mode)"))
	fmt.Println()

	var parts []string
	parts = append(parts, tool)

	for {
		arg := promptInput(reader, "Add flag or argument (leave empty when done):", "")
		if arg == "" {
			break
		}
		parts = append(parts, arg)
	}

	finishCompose(reader, strings.Join(parts, " "))
	return nil
}

// finishCompose shows the composed command and offers actions.
func finishCompose(reader *bufio.Reader, cmd string) string {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Generated command:"))
	fmt.Println()
	fmt.Println("  " + ui.CommandStyle.Render(cmd))
	fmt.Println()

	// Show explanation
	result := explain.Explain(cmd)
	if result.DangerLevel == explain.Destructive {
		fmt.Println(ui.ErrorStyle.Render("  [!] This command has DESTRUCTIVE flags — review carefully"))
	} else if result.DangerLevel == explain.Caution {
		fmt.Println(ui.WarningStyle.Render("  [~] This command has flags that require caution"))
	}

	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			if w.Level == explain.Destructive {
				fmt.Println(ui.ErrorStyle.Render("  [!] " + w.Message))
			}
		}
	}

	fmt.Println()
	fmt.Println(ui.HintStyle.Render("  [c] Copy  [e] Edit  [x] Explain  [q] Quit"))
	fmt.Println()

	return cmd
}

// Prompt helpers

func promptInput(reader *bufio.Reader, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]\n> ", ui.HintStyle.Render(prompt), ui.MetadataStyle.Render(defaultVal))
	} else {
		fmt.Printf("%s\n> ", ui.HintStyle.Render(prompt))
	}

	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptChoice(reader *bufio.Reader, prompt string, choices []string, defaultVal string) string {
	fmt.Println(ui.HintStyle.Render(prompt))
	for i, c := range choices {
		marker := "  "
		if c == defaultVal {
			marker = "> "
		}
		fmt.Printf("%s%d. %s\n", marker, i+1, c)
	}
	fmt.Printf("> ")

	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	if line == "" {
		return defaultVal
	}

	// Accept number input
	for i, c := range choices {
		if line == fmt.Sprintf("%d", i+1) || strings.EqualFold(line, c) {
			return c
		}
	}

	// Try prefix match
	for _, c := range choices {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(line)) {
			return c
		}
	}

	return defaultVal
}

func promptYesNo(reader *bufio.Reader, prompt string, defaultYes bool) bool {
	def := "Y/n"
	if !defaultYes {
		def = "y/N"
	}
	fmt.Printf("%s [%s] ", ui.HintStyle.Render(prompt), def)

	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))

	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}
