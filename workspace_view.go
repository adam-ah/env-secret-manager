package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	canvas  = lipgloss.Color("#EDF2F7")
	surface = lipgloss.Color("#FFFFFF")
	raised  = lipgloss.Color("#E3EDF5")
	ink     = lipgloss.Color("#1B2A3D")
	muted   = lipgloss.Color("#52657C")
	line    = lipgloss.Color("#BDCAD8")
	accent  = lipgloss.Color("#0C7167")
	amber   = lipgloss.Color("#865B11")
	danger  = lipgloss.Color("#AD3045")
)

func text(s string, color lipgloss.Color, bold bool) string {
	return lipgloss.NewStyle().Foreground(color).Background(surface).Bold(bold).Render(s)
}

func canvasText(s string, color lipgloss.Color, bold bool) string {
	return lipgloss.NewStyle().Foreground(color).Background(canvas).Bold(bold).Render(s)
}
func safeLabel(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
func fit(s string, w int) string { return ansi.Truncate(s, max(0, w), "…") }
func fill(s string, w int) string {
	return fit(s, w) + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}
func rule(w int) string { return text(strings.Repeat("─", max(0, w)), line, false) }
func panel(w, h int, focused bool, content string) string {
	border := line
	if focused {
		border = accent
	}
	return restoreBackground(lipgloss.NewStyle().Background(surface).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1).Width(w-2).Height(h-2).MaxWidth(w).MaxHeight(h).Render(content), surface)
}
func label(s string) string { return text(s, muted, false) }

func restoreBackground(s string, color lipgloss.Color) string {
	paint := lipgloss.NewStyle().Background(color).Render(" ")
	prefix, _, _ := strings.Cut(paint, " ")
	if prefix == "" {
		return s
	}
	return strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+prefix) + "\x1b[0m"
}
func shortcut(key, desc string) string {
	return text(key, accent, true) + " " + text(desc, muted, false)
}
func (m workspace) header(w int) string {
	brand := lipgloss.NewStyle().Background(accent).Foreground(canvas).Bold(true).Render(" ◆ ")
	brand += " " + canvasText("secretenv", ink, true) + "  " + canvasText("/  YOUR LOCAL VAULT", muted, false)
	projectCount := len(workspaceProjects(m.cfg))
	projectLabel := "projects"
	if projectCount == 1 {
		projectLabel = "project"
	}
	right := canvasText("●", accent, false) + canvasText("  LOCAL  ·  ", muted, false) + canvasText(fmt.Sprintf("%02d %s", projectCount, projectLabel), ink, false)
	top := brand + strings.Repeat(" ", max(2, w-lipgloss.Width(brand)-lipgloss.Width(right))) + right
	tabs := []string{"1  Vault", "2  Setup", "3  AI access", "4  Help"}
	rendered := []string{}
	for i, title := range tabs {
		st := lipgloss.NewStyle().Padding(0, 2).Foreground(muted).Background(canvas)
		if i == m.page {
			st = st.Foreground(canvas).Background(accent).Bold(true)
		}
		rendered = append(rendered, st.Render(title))
	}
	return fit(top, w) + "\n\n" + strings.Join(rendered, "") + "\n"
}
func (m workspace) footer(w int) string {
	shortcut := func(key, desc string) string {
		return canvasText(key, accent, true) + " " + canvasText(desc, muted, false)
	}
	status := canvasText("●", accent, false) + " " + canvasText("Values stay masked", muted, false) + canvasText("  ·  ~/.secretenv/config.toml", muted, false)
	if m.notice != "" {
		color := accent
		if m.noticeError {
			color = danger
		}
		status = canvasText(m.notice, color, false)
	}
	var hints []string
	switch {
	case m.confirmation != "":
		hints = []string{shortcut("←/→", "Choose action"), shortcut("enter", "Select action"), shortcut("esc", "Cancel")}
	case m.form != "":
		save := "Save secret"
		if m.form == "project" || m.form == "rename-project" {
			save = "Save project"
		}
		if len(m.fields) > 1 {
			hints = append(hints, shortcut("tab", "Next field"))
		}
		hints = append(hints, shortcut("ctrl+s", save), shortcut("esc", "Cancel"))
	case m.searching:
		hints = []string{shortcut("enter", "Finish search"), shortcut("esc", "Clear search")}
	case m.page != 0:
		hints = []string{shortcut("↑/↓", "Scroll"), shortcut("q", "Quit")}
	default:
		hints = []string{shortcut("tab", "Switch pane"), shortcut("n", "New project")}
		if m.selector() != "" {
			hints = append(hints, shortcut("a", "New secret"))
		}
		if m.focus == 0 && m.selector() != "" && m.selector() != globalSelector {
			hints = append(hints, shortcut("e", "Rename project"), shortcut("d", "Delete project"))
		} else if m.focus == 1 && m.selectedSecret() != "" {
			hints = append(hints, shortcut("e", "Edit secret"), shortcut("d", "Delete secret"))
		}
		if m.selector() != "" {
			hints = append(hints, shortcut("/", "Search secrets"))
		}
		hints = append(hints, shortcut("q", "Quit"))
	}
	rows := []string{}
	row := ""
	for _, hint := range hints {
		if row != "" && lipgloss.Width(row)+2+lipgloss.Width(hint) > w {
			rows = append(rows, row)
			row = ""
		}
		if row != "" {
			row += "  "
		}
		row += hint
	}
	if row != "" {
		rows = append(rows, row)
	}
	if len(rows) == 1 || m.notice != "" {
		rows = append([]string{fit(status, w)}, rows...)
	}
	return strings.Join(rows, "\n")
}
func projectTitle(selector string) string {
	if selector == globalSelector {
		return "Global"
	}
	title := filepath.Base(filepath.Clean(selector))
	if title == "." || title == string(filepath.Separator) {
		return selector
	}
	return title
}
func (m workspace) vaultView(w, h int) string {
	sideW := max(21, min(26, w/5))
	left := m.projectPane(sideW, h)
	rest := w - sideW - 1
	if m.selector() == "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", m.emptyVault(rest, h))
	}
	if w >= 104 {
		listW := max(30, min(42, rest/3))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", m.secretPane(listW, h), " ", m.detailsPane(rest-listW-1, h, false))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", m.secretPane(rest, h))
}
func (m workspace) projectPane(w, h int) string {
	cw := w - 4
	heading := text("PROJECTS", ink, true) + text(fmt.Sprintf("  %02d", len(workspaceProjects(m.cfg))), muted, false)
	lines := []string{heading, ""}
	if h < 10 {
		lines = []string{heading}
	}
	names := workspaceProjects(m.cfg)
	available := max(1, h-4)
	if h < 10 {
		available = max(1, h-3)
	}
	start := max(0, m.projectIndex-available+1)
	end := min(len(names), start+available)
	for i := start; i < end; i++ {
		name := safeLabel(projectTitle(names[i]))
		count := fmt.Sprintf("%d", len(scopedProject(m.cfg, names[i]).Env))
		row := "  " + fill(name, cw-4-len(count)) + "  " + count
		if i == m.projectIndex {
			row = "▎ " + fill(name, cw-4-len(count)) + "  " + count
			row = lipgloss.NewStyle().Foreground(accent).Background(raised).Bold(true).Width(cw).Render(row)
		} else {
			row = text(row, muted, false)
		}
		lines = append(lines, fit(row, cw))
	}
	if len(names) == 0 {
		lines = append(lines, label("No projects yet"))
	}
	return panel(w, h, m.focus == 0 && !m.searching, strings.Join(lines, "\n"))
}
func (m workspace) emptyVault(w, h int) string {
	cw := w - 8
	title := text("Your secrets. One place.", ink, true)
	if cw < 28 {
		title = text("Your local vault", ink, true)
	}
	content := text("◇  A CLEAN START", accent, true) + "\n\n" + title + "\n" + lipgloss.NewStyle().Width(cw).Foreground(muted).Render("Keep credentials out of your repos and ready for your tools.") + "\n\n" +
		text("01", accent, true) + "  Create a project\n" + text("02", accent, true) + "  Add your secrets\n" + text("03", accent, true) + "  Use references in shell commands\n\n" +
		text("Create a project to get started.", accent, true)
	return panel(w, h, false, lipgloss.Place(w-4, h-2, lipgloss.Center, lipgloss.Center, content))
}
func (m workspace) secretPane(w, h int) string {
	cw := w - 4
	title := text(safeLabel(projectTitle(m.selector())), ink, true)
	secretCount := len(scopedProject(m.cfg, m.selector()).Env)
	secretLabel := "secrets"
	if secretCount == 1 {
		secretLabel = "secret"
	}
	count := text(fmt.Sprintf("%d %s", secretCount, secretLabel), muted, false)
	heading := fit(title, max(1, cw-lipgloss.Width(count)-2)) + strings.Repeat(" ", max(1, cw-lipgloss.Width(title)-lipgloss.Width(count))) + count
	m.search.Width = max(5, cw-4)
	search := m.search.View()
	description := safeLabel(m.selector())
	if m.selector() == globalSelector {
		description = "Available everywhere · never auto-exported"
	}
	lines := []string{fit(heading, cw), fit(label(description), cw), search, rule(cw)}
	if h < 10 {
		lines = []string{fit(heading, cw), search, rule(cw)}
	}
	names := m.names()
	visible := max(1, h-7)
	if h < 10 {
		visible = max(1, h-5)
	}
	start := max(0, m.secretIndex-visible+1)
	end := min(len(names), start+visible)
	for i := start; i < end; i++ {
		key := safeLabel(names[i])
		mask := "••••••"
		if cw < 31 {
			mask = "••••"
		}
		row := "  " + fill(key, cw-len([]rune(mask))-5) + "  " + text(mask, muted, false)
		if i == m.secretIndex {
			row = "▎ " + fill(key, cw-len([]rune(mask))-5) + "  " + mask
			row = lipgloss.NewStyle().Foreground(ink).Background(raised).Bold(true).Width(cw).Render(row)
		} else {
			row = text("  "+fill(key, cw-len([]rune(mask))-5), muted, false) + "  " + text(mask, muted, false)
		}
		lines = append(lines, fit(row, cw))
	}
	if len(names) == 0 {
		if m.search.Value() != "" {
			lines = append(lines, "", label("No matching secret names."))
		} else {
			lines = append(lines, "", text("No secrets yet.", ink, false))
		}
	}
	for len(lines) < h-3 {
		lines = append(lines, "")
	}
	if len(names) > 0 && h >= 10 {
		lines = append(lines, label(fmt.Sprintf("%d–%d of %d", start+1, end, len(names))))
	}
	return panel(w, h, m.focus == 1 || m.searching, strings.Join(lines, "\n"))
}
func (m workspace) detailsPane(w, h int, compact bool) string {
	cw := w - 4
	name := m.selectedSecret()
	if name == "" {
		return panel(w, h, false, label("SECRET DETAILS")+"\n"+text("Select a secret to inspect its reference.", muted, false))
	}
	reference := safeLabel(m.secretReference())
	if compact {
		return panel(w, h, false, text("REFERENCE", accent, true)+"  "+text(fit(reference, cw-11), ink, false)+"\n"+label("Value hidden"))
	}
	lines := []string{text("SECRET DETAILS", ink, true), label("Use the name. Keep the value private."), "", text(fit(safeLabel(name), cw), accent, true), "", label("VALUE"), text("••••••••••••", muted, false), text("Hidden", accent, false), "", rule(cw), "", label("REFERENCE"), lipgloss.NewStyle().Width(cw).Foreground(ink).Render(reference)}

	if h < 20 {
		lines = []string{text("SECRET DETAILS", ink, true), "", text(fit(safeLabel(name), cw), accent, true), label("Value hidden"), "", rule(cw), "", label("REFERENCE"), lipgloss.NewStyle().Width(cw).Foreground(ink).Render(reference)}
	}
	if h < 14 {
		lines = []string{text("SECRET DETAILS", ink, true), text(fit(safeLabel(name), cw), accent, true), label("Value hidden"), "", label("REFERENCE"), text(fit(reference, cw), ink, false)}
	}
	if h < 10 {
		lines = []string{text("SECRET DETAILS", ink, true), text(fit(safeLabel(name), cw), accent, true), label("Value hidden"), text(fit(reference, cw), ink, false)}
	}
	return panel(w, h, false, strings.Join(lines, "\n"))
}
func (m workspace) formView(w, h int) string {
	title, description := "New project", "Use a project name or an absolute directory path."
	nameLabel := "PROJECT NAME OR PATH"
	if m.form == "rename-project" {
		title = "Rename project"
	}
	if m.form == "secret" || m.form == "edit-secret" {
		title = "New secret"
		if m.form == "edit-secret" {
			title = "Edit secret"
		}
		description = "Project: " + safeLabel(projectTitle(m.selector()))
		nameLabel = "SECRET NAME"
	}
	pw := min(76, w-4)
	cw := pw - 6
	lines := []string{text(title, ink, true), fit(label(description), cw), "", label(nameLabel)}
	for i, input := range m.fields {
		if i == 1 {
			valueLabel := "SECRET VALUE"
			if m.form == "edit-secret" {
				valueLabel = "REPLACEMENT VALUE"
			}
			lines = append(lines, "", label(valueLabel+"  ·  masked"))
		}
		input.Width = max(5, cw-4)
		style := lipgloss.NewStyle().Background(raised).Foreground(ink).Padding(0, 1).Width(cw)
		if i == m.fieldIndex {
			lines = append(lines, text("▎", accent, true)+style.Width(cw-1).Render(input.View()))
		} else {
			lines = append(lines, style.Render(input.View()))
		}
	}
	lines = append(lines, "")
	if m.formError != "" {
		lines = append(lines, fit(text(m.formError, danger, false), cw))
	} else if len(m.fields) > 1 {
		lines = append(lines, fit(label("Empty value saves an empty secret. Existing values are never shown."), cw))
	} else {
		lines = append(lines, fit(label("Examples: my-app · /srv/projects/my-app"), cw))
	}
	content := strings.Join(lines, "\n")
	ph := min(h, lipgloss.Height(content)+2)
	card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Background(surface).Padding(0, 2).Width(pw - 2).MaxHeight(ph).Render(content)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, restoreBackground(card, surface), lipgloss.WithWhitespaceBackground(canvas))
}
func (m workspace) confirmView(w, h int) string {
	pw := min(68, w-4)
	cw := pw - 6
	target := m.selector()
	if m.confirmAction == "delete-secret" {
		target = m.selectedSecret()
	} else if m.confirmAction == "save" && len(m.fields) > 0 {
		target = m.fields[0].Value()
	}
	cancel := lipgloss.NewStyle().Padding(0, 2).Foreground(ink)
	confirm := lipgloss.NewStyle().Padding(0, 2).Foreground(danger)
	if m.confirmYes {
		confirm = confirm.Background(danger).Foreground(canvas).Bold(true)
	} else {
		cancel = cancel.Background(raised).Bold(true)
	}
	action := "Delete"
	color := danger
	if m.confirmAction == "save" {
		action = "Replace"
		color = amber
	}
	content := text(strings.ToUpper(action)+" CONFIRMATION", color, true) + "\n\n" + lipgloss.NewStyle().Width(cw).Foreground(ink).Bold(true).Render(m.confirmation) + "\n" + fit(label(safeLabel(target)), cw) + "\n\n" + label("This change is saved immediately.") + "\n\n" + cancel.Render("Cancel") + "  " + confirm.Render(action)
	card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color).Background(surface).Padding(1, 2).Width(pw - 2).Render(content)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, restoreBackground(card, surface), lipgloss.WithWhitespaceBackground(canvas))
}
