package main

import (
	"fmt"
	"golang.design/x/clipboard"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		split := strings.Split(m.choice, ":")
		clipboard.Write(clipboard.FmtText, []byte(fmt.Sprintf("ssh %s", split[0])))
		return quitTextStyle.Render(fmt.Sprintf("'ssh %s' copied to clipboard", split[0]))
	}
	return "\n" + m.list.View()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	dirname, err := os.UserHomeDir()
	check(err)
	dat, err := os.ReadFile(fmt.Sprintf("%s/.ssh/config", dirname))
	check(err)
	lines := strings.Split(string(dat), "\n")
	hostsIndex := make([]int, 0)
	items := []list.Item{}

	for i, s := range lines {
		if strings.Contains(strings.ToLower(s), "host ") && !strings.Contains(strings.ToLower(s), " *") {
			hostsIndex = append(hostsIndex, i)
			items = append(
				items,
				item(
					strings.Replace(
						fmt.Sprintf("%s: ", strings.ToLower(s)),
						"host ",
						"",
						1),
				),
			)
		} else if strings.Contains(strings.ToLower(s), "hostname ") {
			items[len(items)-1] = item(fmt.Sprintf("%s@%s", items[len(items)-1], strings.Trim(strings.Replace(strings.Trim(strings.ToLower(s), " "), "hostname ", "", 1), " \t")))
		} else if strings.Contains(strings.ToLower(s), "user ") {
			split := strings.Split(items[len(items)-1].FilterValue(), "@")
			host := split[0]

			hostname := ""
			if len(split) > 1 {
				hostname = fmt.Sprintf("@%s", split[1])
			}
			items[len(items)-1] = item(
				fmt.Sprintf(
					"%s%s%s",
					host,
					strings.Trim(strings.Replace(strings.Trim(strings.ToLower(s), " \t"), "user ", "", 1), " \t"),
					hostname,
				),
			)

		}
	}

	hostCount := len(hostsIndex)
	fmt.Print(hostCount)

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select an SSH connection"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
