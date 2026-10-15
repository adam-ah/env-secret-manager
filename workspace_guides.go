package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m workspace) guideView(w, h int) string {
	title, subtitle := "Setup", "Get installed. Make your shell feel at home."
	if m.page == 2 {
		title = "AI access"
		subtitle = "Let tools use a credential by name."
	}
	if m.page == 3 {
		title = "A small field guide"
		subtitle = "Everything you need to move around your vault."
	}
	heading := text(title, ink, true)
	if m.page == 1 {
		heading += "    " + shortcut("z", "Zsh") + "  " + shortcut("b", "Bash") + "  " + text("["+m.shell+"]", accent, true)
	}
	heading += "\n" + fit(text(subtitle, muted, false), w-4)
	m.guide.Width = w - 6
	m.guide.Height = max(1, h-6)
	content := heading + "\n" + rule(w-4) + "\n" + m.guide.View() + "\n" + label(fmt.Sprintf("%3.0f%%  ·  ↑/↓ or PgUp/PgDn to scroll", m.guide.ScrollPercent()*100))
	return panel(w, h, false, content)
}
func (m workspace) guideContent() string {
	w := max(30, m.width-14)
	section := func(n, title, description string) string {
		return text(n+"  ", accent, true) + text(title, ink, true) + "\n" + lipgloss.NewStyle().Width(w).Foreground(muted).Render(description) + "\n"
	}
	code := func(s string) string {
		return lipgloss.NewStyle().Background(raised).Foreground(ink).Padding(1, 2).Width(w).Render(s) + "\n"
	}
	switch m.page {
	case 1:
		return section("01", "Install the binary", "Build from this clone, then place secretenv on your PATH.") + code("./build.sh\nmkdir -p ~/.local/bin\ninstall -m 755 bin/secretenv ~/.local/bin/secretenv") + "\n" +
			section("02", "Add your secrets", "Global is always available and never auto-exports values. Add a secret there, or create a project for automatic directory-based exports. New project names must be safe to type without quotes.") + code("my-app                 matches that folder and its children\n/srv/code/my-app        matches a specific project directory") + "\n" +
			section("03", "Connect your shell · optional", "secretenv install "+m.shell+" prints these instructions without changing files. Add the hook to ~/."+m.shell+"rc, then open a new shell. Project secrets follow the current folder; Global is never auto-injected.") + code(shellHook(m.shell)) + "\n" +
			text("A note for AI sessions", amber, true) + "\n" + lipgloss.NewStyle().Width(w).Foreground(muted).Render("Shell hooks export raw values into child processes. Start AI tools in a separate session without this hook or inherited secrets. Setup instructions here do not modify your shell files.")
	case 2:
		examples := m.examples()
		if examples.name == "" {
			examples = sampleExamples
		}
		return section("01", "Discover references", "list --current returns Global plus the matching project's names and exact references. list returns all projects. Agents use list and exec; never read the config or invoke the internal _hook command.") + code("secretenv list --current\nsecretenv list") + "\n" +
			section("02", envExampleTitle, examples.envDescription()+". PROJECT:KEY becomes KEY, without the project prefix. Add multiple references before --; inherited exported variables and the working directory are preserved, and parent variables do not change.") + code(examples.variableCommand()) + "\n" +
			section("03", envCurlExampleTitle, "sh -c explicitly runs a child shell. Single quotes defer $"+examples.name+" expansion until the child receives the secret.") + code(examples.envCurlCommand()) + "\n" +
			section("04", curlExampleTitle, "Use {KEY} to insert a selected secret into an argument. This example runs curl directly, without a shell. Literal injected values are redacted from stdout and stderr in both examples.") + code(examples.curlCommand()) + "\n" +
			text("Output filtering", amber, true) + "\n" + lipgloss.NewStyle().Width(w).Foreground(muted).Render("Filtering catches literal selected values, not encoded values, files or other channels. Argument values can appear in process listings. The config is plaintext and readable by your OS account.") + "\n\n" +
			text("Agent skill", ink, true) + "\n" + label("skills/secretenv/SKILL.md")
	default:
		return section("MOVE", "Navigate the workspace", "1–4 switch tabs. Tab or ←/→ switches between projects and secrets. ↑/↓ or j/k moves the selection. Enter opens the selected project's secrets. Examples sit below the panes: v copies environment injection, u copies curl using the child env variable, c copies curl using argument substitution.") + "\n" +
			section("EDIT", "Manage your vault", "n creates a project. a creates a secret in the selected section. e renames the selected project or edits the selected secret. d deletes the selection after confirmation. Global cannot be renamed or deleted; its secrets can. / searches secret names; Esc clears the search.") + "\n" +
			section("SAVE", "Edit with confidence", "Tab moves between form fields. Ctrl+S saves; Enter advances or saves the last field. Esc cancels. Replacement and deletion require confirmation. Existing secret values are never loaded into the editor.") + "\n" +
			section("STORE", "One local config", "~/.secretenv/config.toml\nThe editor saves atomically with directory permissions 0700 and file permissions 0600. Values are plaintext on disk.") + "\n" +
			section("CLI", "Keep automation simple", "list prints names and references as JSON. exec injects secrets into child variables and arguments, and filters output. install prints the folder-based shell hook setup without changing files.") + "\n" +
			code(strings.TrimSpace("secretenv help\nsecretenv list\nsecretenv install zsh"))
	}
}
