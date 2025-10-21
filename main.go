package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/mod/modfile"
)

type packageListItem struct {
	title string
	desc  string
}

func (i packageListItem) Title() string       { return i.title }
func (i packageListItem) Description() string { return i.desc }
func (i packageListItem) FilterValue() string { return i.title }

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
	packageList  list.Model
}

func initialModel() model {
	go fetchPackagesFromIndex()

	var installTI = textinput.New()
	installTI.CharLimit = -1
	installTI.Focus()
	installTI.Placeholder = "github.com/..."

	var packList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	packList.Title = "Available packages"
	return model{installInput: installTI, packageList: packList}
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

		case "ctrl+g":
			m.install = !m.install

		case "esc":
			if m.install {
				m.install = false
			}

		case "tab":
			if m.install {
				if m.installInput.Focused() {
					m.installInput.Blur()
					m.packageList.Select(0)
				} else {
					m.installInput.Focus()
				}
			}

		}

	case tea.WindowSizeMsg:
		m.window.width = msg.Width
		m.window.height = msg.Height

		var fmod, err = generatePackageDetails()
		m.err = err

		drawTable(&m, fmod)
		m.rendered = true
	}

	if m.install {
		var cmd tea.Cmd
		m.installInput, cmd = m.installInput.Update(msg)
		return m, cmd
	} else if m.rendered {
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
		var filter = filterGolangPackages(m.installInput.Value())
		var listItems []list.Item

		for _, f := range filter {
			listItems = append(listItems, packageListItem{title: f.Path})
		}
		m.packageList.SetSize(m.window.width-5, m.window.height/2)
		m.packageList.SetItems(listItems)
		m.installInput.Width = m.window.width
		var input = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Width(m.window.width - 5).
			Render(m.installInput.View())

		var output = lipgloss.NewStyle().
			Width(m.window.width - 5).
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center).
			Render(func() string {
				if len(filter) > 0 && len(m.installInput.Value()) > 0 {
					return m.packageList.View()
				}

				return "No package found!"
			}())
		parsedString = fmt.Sprintf("%v\n%v", input, output)
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
