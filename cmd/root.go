/*
Copyright © 2024 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	logger      *log.Logger
	verbose     bool
	githubToken string
	eventCount  int
	since       string
	filterTypes []string // New variable for the filter flag
)

// Define a list of valid event types
var validEventTypes = []string{
	"CommitCommentEvent",
	"CreateEvent",
	"DeleteEvent",
	"ForkEvent",
	"GollumEvent",
	"IssueCommentEvent",
	"IssuesEvent",
	"MemberEvent",
	"PublicEvent",
	"PullRequestEvent",
	"PullRequestReviewEvent",
	"PullRequestReviewCommentEvent",
	"PullRequestReviewThreadEvent",
	"PushEvent",
	"ReleaseEvent",
	"SponsorshipEvent",
	"WatchEvent",
	// Add other event types as needed
}

func parseExtendedDuration(input string) (time.Duration, error) {
	// Regular expression to match duration strings like '1w', '2d', '3h'
	re := regexp.MustCompile(`^(\d+)([smhdw])$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(input))
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration: %s", input)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %v", err)
	}

	unit := matches[2]
	var duration time.Duration
	switch unit {
	case "s":
		duration = time.Duration(value) * time.Second
	case "m":
		duration = time.Duration(value) * time.Minute
	case "h":
		duration = time.Duration(value) * time.Hour
	case "d":
		duration = time.Duration(value) * time.Hour * 24
	case "w":
		duration = time.Duration(value) * time.Hour * 24 * 7
	default:
		return 0, fmt.Errorf("unknown unit in duration: %s", unit)
	}
	return duration, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitfamous <username>",
	Short: "Github Event Tracker TUI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
		// Retrieve GitHub token
		if githubToken == "" {
			githubToken = os.Getenv("GITHUB_TOKEN")
			if githubToken == "" {
				githubToken = os.Getenv("GITHUB_API_TOKEN")
			}
		}
		if githubToken == "" {
			logger.Error("Github API token is required")
			os.Exit(1)
		}
		var sinceDuration time.Duration
		if since == "" {
			sinceDuration = 0
		} else {
			var err error
			sinceDuration, err = parseExtendedDuration(since)
			if err != nil {
				logger.Error("parsing since duration", "error", err)
				os.Exit(1)
			}
		}
		for _, f := range filterTypes {
			if !slices.Contains(validEventTypes, f) {
				logger.Warn("Invalid event type in --filter:", f)
			}
		}

		// Start the TUI application
		p := tea.NewProgram(initialModel(args[0], githubToken, eventCount, sinceDuration, filterTypes), tea.WithAltScreen())
		// p := tea.NewProgram(initialModel(args[0], githubToken))
		if m, err := p.Run(); err != nil {
			logger.Error("running gitfamous", "error", err)
			os.Exit(1)
		} else {
			if m, ok := m.(model); ok {
				if m.err != nil {
					logger.Error(m.err)
					return
				}
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Override the default error level style.
	styles := log.DefaultStyles()
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("ERROR").
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("204")).
		Foreground(lipgloss.Color("0"))
	// Add a custom style for key `err`
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	logger = log.New(os.Stderr)
	logger.SetStyles(styles)
	// Define CLI flags
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "Verbose output")
	rootCmd.Flags().StringVarP(&githubToken, "api", "t", "", "Github API Token")
	rootCmd.Flags().IntVarP(&eventCount, "count", "c", 0, "Number of events to fetch")
	rootCmd.Flags().StringVarP(&since, "since", "s", "", "Limit events to those after the specified amount of time (e.g. 1h, 1d, 1w)")
	rootCmd.Flags().StringSliceVarP(&filterTypes, "filter", "f", nil, "Comma-separated list of event types to display")
}
