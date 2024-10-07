package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/google/go-github/v66/github"
	"golang.org/x/term"
)

type Actor struct {
	Login     string
	AvatarURL string
}

type Repo struct {
	Name string
	URL  string
}

type eventItem struct {
	Date        string
	Type        string
	Actor       *Actor
	Repository  *Repo
	Description string
}

type model struct {
	username    string
	apiToken    string
	events      []eventItem
	table       table.Model
	err         error
	count       int
	since       time.Duration
	filterTypes []string // New field for filter criteria
	tableHeight int
}

var baseTableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func initialModel(username, apiToken string, count int, since time.Duration, filterTypes []string) model {
	return model{
		username:    username,
		apiToken:    apiToken,
		count:       count,
		since:       since,
		filterTypes: filterTypes,
	}
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
		events, err := fetchEvents(m.username, m.apiToken, m.count, m.since, m.filterTypes)
		return fetchEventsMsg{
			events: events,
			err:    err,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case fetchEventsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.events = msg.events

		maxColWidths := map[string][]int{
			"Date":        {},
			"Repository":  {},
			"Description": {},
		}
		// Create table rows
		var rows []table.Row
		for _, event := range m.events {
			maxColWidths["Date"] = append(maxColWidths["Date"], len(event.Date))
			maxColWidths["Repository"] = append(maxColWidths["Repository"], len(event.Repository.Name))
			maxColWidths["Description"] = append(maxColWidths["Description"], len(event.Description))
			row := table.Row{event.Date, event.Repository.Name, event.Description}
			rows = append(rows, row)
		}

		// Get terminal width
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width = 80 // Default width if there's an error
		}

		// Calculate max widths of columns based on content
		dateWidth := slices.Max(maxColWidths["Date"])
		repoWidth := slices.Max(maxColWidths["Repository"])

		// Calculate spacing (adjust based on your table's formatting)
		spacing := 4 // Adjust this value based on actual padding and separators in your table

		// Define the desired right padding (in number of spaces)
		rightPadding := spacing * 3 // Adjust this value as needed

		// Calculate Description column width to fill remaining terminal width minus right padding
		descWidth := width - dateWidth - repoWidth - spacing - rightPadding
		if descWidth < 20 { // Set a minimum width for Description
			descWidth = 20
		}

		// Define table columns with calculated widths
		columns := []table.Column{
			{Title: "Date", Width: dateWidth + spacing},
			{Title: "Repository", Width: repoWidth + spacing},
			{Title: "Description", Width: descWidth},
		}

		m.tableHeight = len(rows) + 1
		if m.tableHeight > 30 {
			m.tableHeight = 30
		}

		// Initialize table model with updated columns
		m.table = table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.tableHeight),
		)

		// Optional: Customize table styles
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
		m.table.SetStyles(s)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		// case "esc":
		// 	if m.table.Focused() {
		// 		m.table.Blur()
		// 	} else {
		// 		m.table.Focus()
		// 	}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			m.handleEnterKey()
		}
	}

	// Update the table with any unhandled messages
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if len(m.events) == 0 {
		return "Loading events...\n"
	}

	return baseTableStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n"
}

func fetchEvents(username, api string, count int, since time.Duration, filterTypes []string) ([]eventItem, error) {
	ctx := context.Background()

	client := github.NewClient(nil).WithAuthToken(api)

	opt := &github.ListOptions{}

	var allEvents []*github.Event
	var fetchedCount int

	for {
		events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, username, true, opt) // true = public only
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			if since > 0 {
				if event.GetCreatedAt().Time.Before(time.Now().Add(-since)) {
					break
				}
			}
			if len(filterTypes) > 0 {
				if !slices.Contains(filterTypes, event.GetType()) {
					continue
				}
			}
			allEvents = append(allEvents, event)
			fetchedCount++
			if 0 < count && fetchedCount >= count {
				break
			}
		}

		if (0 < count && fetchedCount >= count) || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Process the events
	var eventItems []eventItem
	for _, event := range allEvents {
		item := eventItem{
			Date:        humanize.Time(event.GetCreatedAt().Time),
			Type:        event.GetType(),
			Actor:       &Actor{Login: event.GetActor().GetLogin(), AvatarURL: event.GetActor().GetAvatarURL()},
			Repository:  &Repo{Name: event.GetRepo().GetName(), URL: event.GetRepo().GetURL()},
			Description: getEventDescription(event),
		}
		eventItems = append(eventItems, item)
	}

	if len(eventItems) == 0 {
		return nil, fmt.Errorf("no events found for user %s (since %s)", username, since)
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
	case "CommitCommentEvent":
		if commitCommentEvent, ok := payload.(*github.CommitCommentEvent); ok {
			return fmt.Sprintf("󰆃 Commit comment on #%d: %s", commitCommentEvent.GetComment().GetPosition(), commitCommentEvent.GetComment().GetBody())
		}
	case "CreateEvent":
		if createEvent, ok := payload.(*github.CreateEvent); ok {
			var icon string
			switch *createEvent.RefType {
			case "branch":
				icon = "󱓊"
			case "tag":
				icon = "󱈢"
			case "repository":
				icon = "󰳏"
			default:
				icon = ""
			}
			return fmt.Sprintf("%s Created %s (%s)", icon, createEvent.GetRefType(), createEvent.GetRef())
		}
	case "DeleteEvent":
		if deleteEvent, ok := payload.(*github.DeleteEvent); ok {
			return fmt.Sprintf("󰆴 Deleted %s (%s)", deleteEvent.GetRefType(), deleteEvent.GetRef())
		}
	case "ForkEvent":
		if _, ok := payload.(*github.ForkEvent); ok {
			return " Forked repository"
		}
	case "GollumEvent":
		if _, ok := payload.(*github.GollumEvent); ok {
			return fmt.Sprintf("󰷉 Wiki page event")
		}
	case "IssueCommentEvent":
		if payload, ok := payload.(*github.IssueCommentEvent); ok {
			return fmt.Sprintf("󰅽 Issue comment on #%d: %#v", payload.GetIssue().GetNumber(), payload.GetComment().GetBody())
		}
	case "IssuesEvent":
		if payload, ok := payload.(*github.IssuesEvent); ok {
			return fmt.Sprintf("󱋄 Issue #%d %s: %s", payload.GetIssue().GetNumber(), payload.GetAction(), payload.GetIssue().GetTitle())
		}
	case "MemberEvent":
		if payload, ok := payload.(*github.MemberEvent); ok {
			return fmt.Sprintf(" Member %s %s", payload.GetMember().GetLogin(), payload.GetAction())
		}
	case "PublicEvent":
		if payload, ok := payload.(*github.PublicEvent); ok {
			return fmt.Sprintf("👀 Repository %s made public", payload.GetRepo().GetName())
		}
	case "PullRequestEvent":
		if payload, ok := payload.(*github.PullRequestEvent); ok {
			return fmt.Sprintf(" PR #%d %s", payload.GetNumber(), payload.GetAction())
		}
	case "PullRequestReviewEvent":
		if payload, ok := payload.(*github.PullRequestReviewEvent); ok {
			return fmt.Sprintf("  PR review on #%d", payload.GetPullRequest().GetNumber())
		}
	case "PullRequestReviewCommentEvent":
		if payload, ok := payload.(*github.PullRequestReviewCommentEvent); ok {
			return fmt.Sprintf("   PR review comment on #%d", payload.GetPullRequest().GetNumber())
		}
	case "PullRequestReviewThreadEvent":
		if payload, ok := payload.(*github.PullRequestReviewThreadEvent); ok {
			return fmt.Sprintf("  PR review thread on #%d", payload.GetPullRequest().GetNumber())
		}
	case "PushEvent":
		if pushEvent, ok := payload.(*github.PushEvent); ok {
			if len(pushEvent.GetCommits()) > 0 {
				return fmt.Sprintf(" Pushed %d commit(s) to %s: %#v", len(pushEvent.GetCommits()), pushEvent.GetRef(), pushEvent.GetCommits()[0].GetMessage())
			}
		}
	case "ReleaseEvent":
		if payload, ok := payload.(*github.ReleaseEvent); ok {
			return fmt.Sprintf("󰎔 Released %s", payload.GetRelease().GetName())
		}
	case "SponsorshipEvent":
		if payload, ok := payload.(*github.SponsorshipEvent); ok {
			return fmt.Sprintf(" Sponsorship event on %s", payload.GetRepository())
		}
	case "WatchEvent":
		if _, ok := payload.(*github.WatchEvent); ok {
			return "⭐️ Starred repository"
		}
	default:
		return fmt.Sprintf("%#v", payload)
	}
	return ""
}

// Function to open a URL in the default browser
func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func (m *model) handleEnterKey() {
	selectedRow := m.table.SelectedRow()
	if selectedRow == nil {
		return
	}

	repoURL := "https://github.com/" + selectedRow[1]

	// Validate URL
	if _, err := url.ParseRequestURI(repoURL); err != nil {
		log.Printf("Invalid URL: %v", err)
		return
	}

	// Open the URL in the default browser
	if err := openURL(repoURL); err != nil {
		log.Printf("Failed to open URL: %v", err)
	}
}
