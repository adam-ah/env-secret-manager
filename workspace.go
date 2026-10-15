package main

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type workspace struct {
	cfg                                    Config
	width, height                          int
	page, focus, projectIndex, secretIndex int
	search                                 textinput.Model
	searching                              bool
	guide                                  viewport.Model
	shell                                  string
	notice                                 string
	noticeError                            bool
	form                                   string
	fields                                 []textinput.Model
	fieldIndex                             int
	originalName                           string
	formError                              string
	confirmation                           string
	confirmAction                          string
	confirmYes                             bool
}

func newWorkspace(cfg Config) workspace {
	if cfg.Global.Env == nil {
		cfg.Global.Env = map[string]string{}
	}
	search := textinput.New()
	search.Prompt = "⌕  "
	search.Placeholder = "Filter secret names…"
	search.CharLimit = 120
	search.TextStyle = lipgloss.NewStyle().Foreground(ink).Background(surface)
	search.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted).Background(surface)
	search.PromptStyle = lipgloss.NewStyle().Foreground(accent).Background(surface)
	m := workspace{cfg: cfg, width: 100, height: 32, search: search, shell: "zsh", guide: viewport.New(80, 20)}
	if cwd, err := os.Getwd(); err == nil {
		if match := matchingProject(cfg, cleanPath(cwd)); match != nil {
			for i, name := range workspaceProjects(cfg) {
				if name == match.Selector {
					m.projectIndex = i
					break
				}
			}
		}
	}
	m.resizeGuide()
	return m
}
func (m workspace) Init() tea.Cmd { return nil }
func (m workspace) selector() string {
	names := workspaceProjects(m.cfg)
	if len(names) == 0 {
		return ""
	}
	return names[min(m.projectIndex, len(names)-1)]
}
func (m workspace) names() []string {
	names := secretNames(scopedProject(m.cfg, m.selector()))
	query := strings.ToLower(m.search.Value())
	if query == "" {
		return names
	}
	filtered := []string{}
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), query) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
func (m workspace) selectedSecret() string {
	names := m.names()
	if len(names) == 0 {
		return ""
	}
	return names[min(m.secretIndex, len(names)-1)]
}
func (m *workspace) resetSecretSelection() { m.secretIndex = 0; m.search.SetValue("") }
func (m *workspace) resizeGuide() {
	m.guide.Width = max(20, m.width-10)
	bodyHeight := m.height - lipgloss.Height(m.header(m.width-4)) - lipgloss.Height(m.footer(m.width-4)) - 3
	m.guide.Height = max(1, bodyHeight-6)
	m.guide.SetContent(m.guideContent())
}
func (m *workspace) flash(message string, failed bool) { m.notice = message; m.noticeError = failed }

func (m workspace) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(copiedCLIExample); ok {
		if result.err != nil {
			m.flash("Clipboard unavailable. Select the command in the example below.", true)
		} else if result.terminal {
			m.flash("Copy sent to your terminal clipboard.", false)
		} else {
			m.flash("CLI example copied. Paste it into your shell.", false)
		}
		return m, nil
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		m.resizeGuide()
		return m, nil
	}
	key, isKey := msg.(tea.KeyMsg)
	if isKey && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.confirmation != "" {
		return m.updateConfirmation(msg)
	}
	if m.form != "" {
		return m.updateForm(msg)
	}
	if m.searching {
		if isKey {
			switch key.String() {
			case "esc":
				m.search.SetValue("")
				m.searching = false
				m.search.Blur()
				m.secretIndex = 0
				return m, nil
			case "enter", "tab":
				m.searching = false
				m.search.Blur()
				m.focus = 1
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.secretIndex = 0
		return m, cmd
	}
	if !isKey {
		return m, nil
	}
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "1", "2", "3", "4":
		m.page = int(key.String()[0] - '1')
		m.notice = ""
		m.resizeGuide()
		m.guide.GotoTop()
		return m, nil
	case "?":
		m.page = 3
		m.resizeGuide()
		m.guide.GotoTop()
		return m, nil
	}
	if m.page != 0 {
		if m.page == 1 && (key.String() == "b" || key.String() == "z") {
			if key.String() == "b" {
				m.shell = "bash"
			} else {
				m.shell = "zsh"
			}
			m.resizeGuide()
			return m, nil
		}
		var cmd tea.Cmd
		m.guide, cmd = m.guide.Update(msg)
		return m, cmd
	}
	switch key.String() {
	case "tab", "shift+tab", "left", "right":
		m.focus = 1 - m.focus
	case "enter":
		m.focus = 1
	case "up", "k":
		if m.focus == 0 {
			m.projectIndex = max(0, m.projectIndex-1)
			m.resetSecretSelection()
		} else {
			m.secretIndex = max(0, m.secretIndex-1)
		}
	case "down", "j":
		if m.focus == 0 {
			m.projectIndex = max(0, min(len(workspaceProjects(m.cfg))-1, m.projectIndex+1))
			m.resetSecretSelection()
		} else {
			m.secretIndex = max(0, min(len(m.names())-1, m.secretIndex+1))
		}
	case "/", "ctrl+f":
		m.focus = 1
		m.searching = true
		return m, m.search.Focus()
	case "esc":
		m.search.SetValue("")
		m.secretIndex = 0
		m.focus = 0
	case "n":
		return m.startForm("project")
	case "c":
		if m.selectedSecret() != "" {
			return m, copyCLIExample(m.cliExample())
		}
	case "v":
		if m.selectedSecret() != "" {
			return m, copyCLIExample(m.variableExample())
		}
	case "u":
		if m.selectedSecret() != "" {
			return m, copyCLIExample(m.envCurlExample())
		}
	case "a":
		if m.selector() == "" {
			return m.startForm("project")
		}
		return m.startForm("secret")
	case "e":
		if m.focus == 0 && m.selector() != globalSelector {
			return m.startForm("rename-project")
		}
		if m.focus == 1 && m.selectedSecret() != "" {
			return m.startForm("edit-secret")
		}
	case "d", "delete":
		if m.focus == 0 && m.selector() != globalSelector {
			m.ask("Delete this project and all its secrets?", "delete-project")
		} else if m.focus == 1 && m.selectedSecret() != "" {
			m.ask("Delete this secret?", "delete-secret")
		}
	}
	return m, nil
}

func (m workspace) startForm(kind string) (tea.Model, tea.Cmd) {
	if kind == "rename-project" && m.selector() == globalSelector {
		return m, nil
	}
	m.form = kind
	m.formError = ""
	m.fieldIndex = 0
	m.fields = nil
	m.notice = ""
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "my-app"
	name.CharLimit = 512
	name.TextStyle = lipgloss.NewStyle().Foreground(ink).Background(raised)
	name.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted).Background(raised)
	switch kind {
	case "rename-project":
		name.SetValue(m.selector())
		m.originalName = m.selector()
	case "secret", "edit-secret":
		name.Placeholder = "API_TOKEN"
		name.CharLimit = 256
		if kind == "edit-secret" {
			name.SetValue(m.selectedSecret())
			m.originalName = m.selectedSecret()
		}
	}
	m.fields = append(m.fields, name)
	if kind == "secret" || kind == "edit-secret" {
		value := textinput.New()
		value.Prompt = ""
		value.Placeholder = "Enter secret value"
		value.EchoMode = textinput.EchoPassword
		value.EchoCharacter = '•'
		value.CharLimit = 0
		value.TextStyle = lipgloss.NewStyle().Foreground(accent).Background(raised)
		value.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted).Background(raised)
		m.fields = append(m.fields, value)
	}
	return m, m.fields[0].Focus()
}

func (m workspace) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.closeForm()
			return m, nil
		case "tab", "shift+tab", "up", "down":
			m.fields[m.fieldIndex].Blur()
			if key.String() == "shift+tab" || key.String() == "up" {
				m.fieldIndex = (m.fieldIndex + len(m.fields) - 1) % len(m.fields)
			} else {
				m.fieldIndex = (m.fieldIndex + 1) % len(m.fields)
			}
			return m, m.fields[m.fieldIndex].Focus()
		case "enter":
			if m.fieldIndex < len(m.fields)-1 {
				m.fields[m.fieldIndex].Blur()
				m.fieldIndex++
				return m, m.fields[m.fieldIndex].Focus()
			}
			return m.submitForm()
		case "ctrl+s":
			return m.submitForm()
		}
	}
	var cmd tea.Cmd
	m.fields[m.fieldIndex], cmd = m.fields[m.fieldIndex].Update(msg)
	return m, cmd
}
func (m workspace) submitForm() (tea.Model, tea.Cmd) {
	name := m.fields[0].Value()
	m.fields[0].SetValue(name)
	if m.form == "project" || m.form == "rename-project" {
		if !validProjectName(name) {
			m.formError = "Use letters, numbers, . _ - / :; no leading -; Global is reserved."
			return m, nil
		}
		if _, exists := m.cfg.Projects[name]; exists && !(m.form == "rename-project" && name == m.originalName) {
			m.formError = "A project with this name or path already exists."
			return m, nil
		}
	} else {
		if !validShellName(name) {
			m.formError = "Use letters, numbers and underscores; start with a letter or underscore."
			return m, nil
		}
		if strings.HasPrefix(name, "SECRETENV_ACTIVE_") {
			m.formError = "SECRETENV_ACTIVE_ names are reserved for shell hooks."
			return m, nil
		}
		if _, exists := scopedProject(m.cfg, m.selector()).Env[name]; exists {
			m.ask("Replace the existing value for "+name+"?", "save")
			return m, nil
		}
	}
	m.saveForm()
	return m, nil
}
func (m *workspace) closeForm() {
	for i := range m.fields {
		m.fields[i].SetValue("")
	}
	m.fields = nil
	m.form = ""
	m.formError = ""
	m.originalName = ""
}
func cloneConfig(cfg Config) Config {
	copy := Config{Projects: map[string]Project{}, Global: Project{Env: map[string]string{}}}
	for k, v := range cfg.Global.Env {
		copy.Global.Env[k] = v
	}
	for selector, project := range cfg.Projects {
		env := map[string]string{}
		for k, v := range project.Env {
			env[k] = v
		}
		copy.Projects[selector] = Project{Env: env}
	}
	return copy
}
func (m *workspace) persist(cfg Config, message string) bool {
	if err := saveConfig(cfg); err != nil {
		m.flash("Could not save config. Check directory permissions.", true)
		return false
	}
	m.cfg = cfg
	m.flash(message, false)
	return true
}
func (m *workspace) saveForm() {
	cfg := cloneConfig(m.cfg)
	name := strings.TrimSpace(m.fields[0].Value())
	switch m.form {
	case "project":
		cfg.Projects[name] = Project{Env: map[string]string{}}
	case "rename-project":
		cfg.Projects[name] = cfg.Projects[m.originalName]
		if name != m.originalName {
			delete(cfg.Projects, m.originalName)
		}
	default:
		p := scopedProject(cfg, m.selector())
		if m.form == "edit-secret" && name != m.originalName {
			delete(p.Env, m.originalName)
		}
		p.Env[name] = m.fields[1].Value()
		setScopedProject(&cfg, m.selector(), p)
	}
	kind := m.form
	if !m.persist(cfg, "Saved to your local config.") {
		m.formError = m.notice
		return
	}
	m.closeForm()
	if kind == "project" || kind == "rename-project" {
		for i, p := range workspaceProjects(m.cfg) {
			if p == name {
				m.projectIndex = i
				break
			}
		}
		m.resetSecretSelection()
		m.focus = 1
	} else {
		m.search.SetValue("")
		for i, n := range m.names() {
			if n == name {
				m.secretIndex = i
				break
			}
		}
		m.focus = 1
	}
}
func (m *workspace) ask(question, action string) {
	m.confirmation = question
	m.confirmAction = action
	m.confirmYes = false
}
func (m workspace) updateConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "esc", "n":
		m.confirmation = ""
		return m, nil
	case "left", "right", "tab", "shift+tab":
		m.confirmYes = !m.confirmYes
	case "y":
		m.confirmYes = true
		return m.performConfirmed()
	case "enter":
		if m.confirmYes {
			return m.performConfirmed()
		}
		m.confirmation = ""
	}
	return m, nil
}
func (m workspace) performConfirmed() (tea.Model, tea.Cmd) {
	action := m.confirmAction
	m.confirmation = ""
	if action == "save" {
		m.saveForm()
		return m, nil
	}
	cfg := cloneConfig(m.cfg)
	if action == "delete-project" {
		if m.selector() == globalSelector {
			return m, nil
		}
		delete(cfg.Projects, m.selector())
		if m.persist(cfg, "Project deleted.") {
			m.projectIndex = max(0, min(m.projectIndex, len(workspaceProjects(m.cfg))-1))
			m.resetSecretSelection()
		}
	} else {
		delete(scopedProject(cfg, m.selector()).Env, m.selectedSecret())
		if m.persist(cfg, "Secret deleted.") {
			m.secretIndex = max(0, min(m.secretIndex, len(m.names())-1))
		}
	}
	return m, nil
}

func (m workspace) View() string {
	if m.width < 64 || m.height < 24 {
		return restoreBackground(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, canvasText("secretenv", accent, true)+"\n\n"+canvasText("Give your secrets a little room.", ink, false)+"\n"+canvasText("Resize to at least 64 × 24 · Ctrl+C to quit", muted, false), lipgloss.WithWhitespaceBackground(canvas)), canvas)
	}
	w := m.width - 4
	header := m.header(w)
	footer := m.footer(w)
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 3
	var body string
	switch {
	case m.confirmation != "":
		body = m.confirmView(w, bodyHeight)
	case m.form != "":
		body = m.formView(w, bodyHeight)
	case m.page == 0:
		if m.selectedSecret() != "" {
			example := m.cliExampleView(w)
			body = lipgloss.JoinVertical(lipgloss.Left, m.vaultView(w, bodyHeight-lipgloss.Height(example)-1), "", example)
		} else {
			body = m.vaultView(w, bodyHeight)
		}
	default:
		body = m.guideView(w, bodyHeight)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, "", footer)
	return restoreBackground(lipgloss.NewStyle().Background(canvas).Foreground(ink).Width(m.width).Height(m.height).Padding(1, 2).MaxWidth(m.width).MaxHeight(m.height).Render(content), canvas)
}
