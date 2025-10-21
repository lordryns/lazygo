package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/mod/modfile"
)

type gopackage struct {
	path       string
	version    string
	isIndirect bool
}

type gomod struct {
	goVersion string
	packages  []gopackage
}

type dimension struct {
	width  int
	height int
}

type model struct {
	packageCount int
	window       dimension
	ptable       table.Model
	rendered     bool
	err          error
	install      bool
	installInput textinput.Model
}

func initialModel() model {
	var installTI = textinput.New()
	installTI.CharLimit = -1
	installTI.Focus()
	installTI.Placeholder = "github.com/..."
	return model{installInput: installTI}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "i":
			m.install = !m.install

		}

	case tea.WindowSizeMsg:
		m.window.width = msg.Width
		m.window.height = msg.Height

		var fmod, err = generatePackageDetails()
		m.err = err

		drawTable(&m, fmod)
		m.rendered = true
	}

	if m.rendered {
		var cmd tea.Cmd
		m.ptable, cmd = m.ptable.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var parsedString string

	if m.window.width < 60 || m.window.height < 30 {
		parsedString = lipgloss.NewStyle().Width(m.window.width).Height(m.window.height).Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("Terminal size too small!\nMust be at least 60, 30\nCurrent: %v, %v", m.window.width, m.window.height))

		return parsedString
	}
	var fmod, err = generatePackageDetails()
	if err != nil {
		parsedString = lipgloss.NewStyle().Width(m.window.width).Height(m.window.height).Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("An error occured! err: %v", err.Error()))

		return parsedString
	}
	var projectVersion = lipgloss.NewStyle().
		Width(m.window.width).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("Go version: %v", fmod.goVersion))

	parsedString = fmt.Sprintf("%v\n%v", projectVersion, m.ptable.View())

	if m.install {
		m.installInput.Width = m.window.width / 2
		parsedString = lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.installInput.View())
	}
	return parsedString
}

func drawTable(m *model, fmod gomod) {
	columns := []table.Column{
		{Title: "Package", Width: m.window.width / 2},
		{Title: "Version", Width: (m.window.width / 2) / 2},
		{Title: "Indirect", Width: (m.window.width / 2) / 2},
	}

	var rows []table.Row
	for _, pack := range fmod.packages {
		rows = append(rows, table.Row{pack.path, pack.version, fmt.Sprintf("%v", pack.isIndirect)})
	}

	m.ptable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.window.height/2),
	)

	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	style.Selected = style.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	m.ptable.SetStyles(style)
}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		panic(err)
	}

}

func generatePackageDetails() (gomod, error) {
	var fbytes, ferr = os.ReadFile("go.mod")
	if ferr != nil {
		return gomod{}, ferr
	}
	var fmod, perr = modfile.Parse("go.mod", fbytes, nil)
	if perr != nil {
		return gomod{}, perr
	}

	var packages []gopackage
	for _, pack := range fmod.Require {
		packages = append(packages, gopackage{path: pack.Mod.Path, version: pack.Mod.Version, isIndirect: pack.Indirect})
	}
	return gomod{goVersion: fmod.Go.Version, packages: packages}, nil
}
