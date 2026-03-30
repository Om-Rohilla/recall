package compose

import (
	"bufio"
	"fmt"
	"strings"
)

type Prompter struct {
	reader *bufio.Reader
	styles StyleAdapter
}

type StyleAdapter interface {
	Title(s string) string
	Metadata(s string) string
	Command(s string) string
	Hint(s string) string
	Error(s string) string
	Warning(s string) string
}

func New(reader *bufio.Reader, styles StyleAdapter) *Prompter {
	return &Prompter{reader: reader, styles: styles}
}

func (p *Prompter) PromptInput(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]\n> ", p.styles.Hint(prompt), p.styles.Metadata(defaultVal))
	} else {
		fmt.Printf("%s\n> ", p.styles.Hint(prompt))
	}

	line, _ := p.reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func (p *Prompter) PromptChoice(prompt string, choices []string, defaultVal string) string {
	fmt.Println(p.styles.Hint(prompt))
	for i, c := range choices {
		marker := "  "
		if c == defaultVal {
			marker = "> "
		}
		fmt.Printf("%s%d. %s\n", marker, i+1, c)
	}
	fmt.Printf("> ")

	line, _ := p.reader.ReadString('\n')
	line = strings.TrimSpace(line)

	if line == "" {
		return defaultVal
	}

	for i, c := range choices {
		if line == fmt.Sprintf("%d", i+1) || strings.EqualFold(line, c) {
			return c
		}
	}

	for _, c := range choices {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(line)) {
			return c
		}
	}

	return defaultVal
}

func (p *Prompter) PromptYesNo(prompt string, defaultYes bool) bool {
	def := "Y/n"
	if !defaultYes {
		def = "y/N"
	}
	fmt.Printf("%s [%s] ", p.styles.Hint(prompt), def)

	line, _ := p.reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))

	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

// ShellEscape escapes single quotes in user input for safe embedding in shell commands.
func ShellEscape(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}
