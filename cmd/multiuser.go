package cmd

import (
	"os"
	"slices"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Multi-user model functions
func initialMultiUserModel(config *Config) multiUserModel {
	tabs := make([]userTab, len(config.Users))
	for i, user := range config.Users {
		token := user.Token
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
			if token == "" {
				token = os.Getenv("GITHUB_API_TOKEN")
			}
		}

		tabs[i] = userTab{
			username:    user.Username,
			apiToken:    token,
			state:       TabLoading,
			count:       config.DefaultSettings.Count,
			filterTypes: config.DefaultSettings.Filter,
		}

		if config.DefaultSettings.Since != "" {
			if duration, err := parseExtendedDuration(config.DefaultSettings.Since); err == nil {
				tabs[i].since = duration
			}
		}
	}

	return multiUserModel{
		tabs:      tabs,
		activeTab: 0,
		config:    config,
	}
}

func (m multiUserModel) Init() tea.Cmd {
	// Start fetching events for all users simultaneously
	var cmds []tea.Cmd
	for i := range m.tabs {
		cmds = append(cmds, m.fetchEventsForUser(i))
	}
	return tea.Batch(cmds...)
}

func (m multiUserModel) fetchEventsForUser(userIndex int) tea.Cmd {
	return func() tea.Msg {
		tab := m.tabs[userIndex]
		events, err := fetchEvents(tab.username, tab.apiToken, tab.count, tab.since, tab.filterTypes)
		return fetchEventsForUserMsg{
			userIndex: userIndex,
			events:    events,
			err:       err,
		}
	}
}

func (m multiUserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchEventsForUserMsg:
		if msg.userIndex < len(m.tabs) {
			if msg.err != nil {
				m.tabs[msg.userIndex].err = msg.err
				m.tabs[msg.userIndex].state = TabError
			} else {
				m.tabs[msg.userIndex].events = msg.events
				m.tabs[msg.userIndex].state = TabLoaded
				m.tabs[msg.userIndex] = m.setupTableForTab(m.tabs[msg.userIndex])
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left", "j":
			if m.activeTab > 0 {
				m.activeTab--
			}
			return m, nil
		case "right", "l":
			if m.activeTab < len(m.tabs)-1 {
				m.activeTab++
			}
			return m, nil
		case "enter":
			if m.activeTab < len(m.tabs) && m.tabs[m.activeTab].state == TabLoaded {
				m.handleEnterKeyForTab(m.activeTab)
			}
			return m, nil
		}
	}

	// Update the active tab's table
	if m.activeTab < len(m.tabs) && m.tabs[m.activeTab].state == TabLoaded {
		var cmd tea.Cmd
		m.tabs[m.activeTab].table, cmd = m.tabs[m.activeTab].table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m multiUserModel) setupTableForTab(tab userTab) userTab {
	maxColWidths := map[string][]int{
		"Date":        {},
		"Repository":  {},
		"Description": {},
	}

	var rows []table.Row
	for _, event := range tab.events {
		maxColWidths["Date"] = append(maxColWidths["Date"], len(event.Date))
		maxColWidths["Repository"] = append(maxColWidths["Repository"], len(event.Repository.Name))
		maxColWidths["Description"] = append(maxColWidths["Description"], len(event.Description))
		row := table.Row{event.Date, event.Repository.Name, event.Description}
		rows = append(rows, row)
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	dateWidth := slices.Max(maxColWidths["Date"])
	repoWidth := slices.Max(maxColWidths["Repository"])
	spacing := 4
	rightPadding := spacing * 3
	descWidth := width - dateWidth - repoWidth - spacing - rightPadding - 20 // Extra space for tab bar
	if descWidth < 20 {
		descWidth = 20
	}

	columns := []table.Column{
		{Title: "Date", Width: dateWidth + spacing},
		{Title: "Repository", Width: repoWidth + spacing},
		{Title: "Description", Width: descWidth},
	}

	tab.tableHeight = len(rows) + 1
	if tab.tableHeight > 25 { // Smaller height to accommodate tab bar
		tab.tableHeight = 25
	}

	tab.table = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tab.tableHeight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Foreground(lipgloss.Color("63")).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tab.table.SetStyles(s)

	return tab
}

func (m multiUserModel) View() string {
	// Render tab bar
	var tabBar []string
	for i, tab := range m.tabs {
		style := inactiveTabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}

		tabText := tab.username
		switch tab.state {
		case TabLoading:
			tabText += " (loading...)"
		case TabError:
			tabText += " (error)"
		}

		tabBar = append(tabBar, style.Render(tabText))
	}

	tabBarView := lipgloss.JoinHorizontal(lipgloss.Top, tabBar...)

	// Render active tab content
	if m.activeTab >= len(m.tabs) {
		return tabBarView + "\n\nNo active tab"
	}

	activeTab := m.tabs[m.activeTab]
	switch activeTab.state {
	case TabLoading:
		return tabBarView + "\n\nLoading events for " + activeTab.username + "..."
	case TabError:
		errorMsg := "Error loading events for " + activeTab.username
		if activeTab.err != nil {
			errorMsg += ": " + activeTab.err.Error()
		}
		return tabBarView + "\n\n" + errorMsg
	case TabLoaded:
		if len(activeTab.events) == 0 {
			return tabBarView + "\n\nNo events found for " + activeTab.username
		}
		return tabBarView + "\n\n" + baseTableStyle.Render(activeTab.table.View()) + "\n  " + activeTab.table.HelpView() + "\n"
	default:
		return tabBarView + "\n\nUnknown state"
	}
}

func (m multiUserModel) handleEnterKeyForTab(tabIndex int) {
	if tabIndex >= len(m.tabs) {
		return
	}

	tab := m.tabs[tabIndex]
	selectedRow := tab.table.SelectedRow()
	if selectedRow == nil {
		return
	}

	repoURL := "https://github.com/" + selectedRow[1]

	if err := openURL(repoURL); err != nil {
		log.WithError(err).Errorf("Failed to open URL: %s", repoURL)
	}
}
