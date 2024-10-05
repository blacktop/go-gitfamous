package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/google/go-github/v65/github"
)

type eventItem struct {
	Date        string
	Type        string
	Repository  string
	Description string
}

type model struct {
	username string
	apiToken string
	events   []eventItem
	quitting bool
	err      error
}

var quitKeys = key.NewBinding(
	key.WithKeys("q", "esc", "ctrl+c"),
	key.WithHelp("", "press q to quit"),
)

func initialModel(username, apiToken string) model {
	return model{username: username, apiToken: apiToken}
}

func (m model) Init() tea.Cmd {
	return m.fetchEventsCmd()
}

// Message type for fetched events
type fetchEventsMsg struct {
	events []eventItem
	err    error
}

func (m model) fetchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		events, err := fetchEvents(m.username, m.apiToken)
		return fetchEventsMsg{
			events: events,
			err:    err,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case fetchEventsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.events = msg.events
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, quitKeys) {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// If events are not yet loaded, show a loading message
	if len(m.events) == 0 {
		return "Loading events...\n"
	}

	// Define table headers
	headers := []string{"Date", "Type", "Repository", "Description"}

	// Collect rows of data
	var rows [][]string
	for _, event := range m.events {
		row := []string{event.Date, event.Type, event.Repository, event.Description}
		rows = append(rows, row)
	}

	// Define table styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	rowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// Calculate column widths
	colWidths := []int{15, 17, 38, 100} // Adjust as needed

	// Build the table
	var table string
	// Create header
	for i, h := range headers {
		table += headerStyle.Render(padRight(h, " ", colWidths[i]))
		if i < len(headers)-1 {
			table += "  "
		}
	}
	table += "\n"

	// Create rows
	for _, row := range rows {
		for i, col := range row {
			table += rowStyle.Render(padRight(col, " ", colWidths[i]))
			if i < len(row)-1 {
				table += "  "
			}
		}
		table += "\n"
	}

	return table
}

// Helper function to pad strings to a specific width
func padRight(str, pad string, length int) string {
	for len(str) < length {
		str += pad
	}
	if len(str) > length {
		str = str[:length-3] + "..." // Truncate and add ellipsis if too long
	}
	return str
}

func fetchEvents(username, api string) ([]eventItem, error) {
	client := github.NewClient(nil).WithAuthToken(api)
	// Fetch the events for the specified user
	events, _, err := client.Activity.ListEventsPerformedByUser(context.Background(), username, true, nil) // true = public only
	if err != nil {
		return nil, err
	}

	// Process the events and extract relevant information
	var eventItems []eventItem
	for _, event := range events {
		item := eventItem{
			Date:        humanize.Time(event.GetCreatedAt().Time),
			Type:        event.GetType(),
			Repository:  event.GetRepo().GetName(),
			Description: getEventDescription(event),
		}
		eventItems = append(eventItems, item)
	}

	return eventItems, nil
}

// Helper function to get a description based on event type
func getEventDescription(event *github.Event) string {
	payload, err := event.ParsePayload()
	if err != nil {
		return fmt.Sprintf("[ERROR] %v", err)
	}
	switch *event.Type {
	case "CreateEvent":
		if createEvent, ok := payload.(*github.CreateEvent); ok {
			return fmt.Sprintf("Created %s (%s)", *createEvent.Ref, *createEvent.RefType)
		}
	case "DeleteEvent":
		if deleteEvent, ok := payload.(*github.DeleteEvent); ok {
			return fmt.Sprintf("Deleted %s (%s)", *deleteEvent.Ref, *deleteEvent.RefType)
		}
	case "PushEvent":
		if pushEvent, ok := payload.(*github.PushEvent); ok {
			if len(pushEvent.Commits) > 0 {
				return fmt.Sprintf("Pushed %d commit(s): %s", len(pushEvent.Commits), pushEvent.Commits[0].GetMessage())
			}
		}
	case "PullRequestEvent":
		if payload, ok := payload.(*github.PullRequestEvent); ok {
			return fmt.Sprintf("PR #%d %s", payload.GetNumber(), payload.GetAction())
		}
	case "IssuesEvent":
		if payload, ok := payload.(*github.IssuesEvent); ok {
			return fmt.Sprintf("Issue #%d %s: %s", payload.GetIssue().GetNumber(), payload.GetAction(), payload.GetIssue().GetTitle())
		}
	case "IssueCommentEvent":
		if payload, ok := payload.(*github.IssueCommentEvent); ok {
			return fmt.Sprintf("Issue comment on #%d: %s", payload.GetIssue().GetNumber(), payload.GetComment().GetBody())
		}
	case "PullRequestReviewEvent":
		if payload, ok := payload.(*github.PullRequestReviewEvent); ok {
			return fmt.Sprintf("PR review on #%d", payload.GetPullRequest().GetNumber())
		}
	case "PullRequestReviewCommentEvent":
		if payload, ok := payload.(*github.PullRequestReviewCommentEvent); ok {
			return fmt.Sprintf("PR review comment on #%d", payload.GetPullRequest().GetNumber())
		}
	case "WatchEvent":
		if payload, ok := payload.(*github.WatchEvent); ok {
			return fmt.Sprintf("Watched repository %s", payload.GetAction())
		}
	case "StarEvent":
		if payload, ok := payload.(*github.StarEvent); ok {
			return fmt.Sprintf("Starred repository %s", payload.GetRepo().GetName())
		}
	case "ForkEvent":
		if payload, ok := payload.(*github.ForkEvent); ok {
			return fmt.Sprintf("Forked repository %s", payload.GetRepo().GetName())
		}
	// Add more cases for different event types as needed
	default:
		return ""
	}
	return ""
}
