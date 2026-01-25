package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Config
const (
	daemonURL      = "http://localhost:8090"
	pollRate       = time.Second
	maxEvents      = 20
	viewportHeight = 20
)

// Styles
var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dotStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	mainStyle   = lipgloss.NewStyle().MarginLeft(1)
	statusStyle = lipgloss.NewStyle().Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	// Layout styles
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Width(100)

	paneStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Width(100)

	// Event styles
	eventTimeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(20)
	eventTypeStyle  = lipgloss.NewStyle().Width(25).Bold(true)
	eventAgentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")) // Purple

	denyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	approveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Blue
)

// API Types (mirrored from pkg/store and pkg/api to avoid CGO deps)

type Event struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	TsEvent    time.Time       `json:"ts_event"`
	Dimensions EventDimensions `json:"dimensions"`
	Payload    json.RawMessage `json:"payload"`
}

type EventDimensions struct {
	AgentID    string `json:"agent_id"`
	IdentityID string `json:"identity_id"`
	WorkloadID string `json:"workload_id"`
	ScopeID    string `json:"scope_id"`
}

// Identity represents the identity structure from the API
type Identity struct {
	IdentityID string                 `json:"identity_id"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type tickMsg time.Time

type dataMsg struct {
	events     []Event
	identities map[string]Identity
	err        error
}

type model struct {
	spinner    spinner.Model
	viewport   viewport.Model
	events     []Event
	identities map[string]Identity
	err        error
	ready      bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(100, viewportHeight)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	return model{
		spinner:    s,
		events:     []Event{},
		identities: make(map[string]Identity),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchData(),
		tick(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Pass key messages to viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tickMsg:
		cmds = append(cmds, fetchData(), tick())

	case dataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.events = msg.events
			m.identities = msg.identities
			m.updateViewportContent()
		}

		if !m.ready {
			m.ready = true
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				PaddingRight(2)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateViewportContent() {
	var sb strings.Builder

	// Sort events by time (newest first)
	// API usually returns them sorted, but good to be safe if we were merging
	// For display, we usually want newest at bottom for logs, or top?
	// The prompt says "scrolling viewport", usually implies log style.
	// Let's print them in the order received, which is likely desc or asc.
	// If API returns limit=20, it's probably the last 20.

	for _, e := range m.events {
		ts := e.TsEvent.Format("15:04:05")

		// Colorize based on event type
		var typeStr string
		switch {
		case strings.Contains(e.EventType, "denied") || strings.Contains(e.EventType, "error"):
			typeStr = denyStyle.Render(e.EventType)
		case strings.Contains(e.EventType, "approved") || strings.Contains(e.EventType, "registered"):
			typeStr = approveStyle.Render(e.EventType)
		default:
			typeStr = infoStyle.Render(e.EventType)
		}

		// Format: [TIMESTAMP] [TYPE] Agent: [AGENT_ID]
		line := fmt.Sprintf("%s %s %s\n",
			eventTimeStyle.Render(ts),
			typeStr,
			eventAgentStyle.Render(fmt.Sprintf("Agent: %s", e.Dimensions.AgentID)),
		)
		sb.WriteString(line)
	}

	m.viewport.SetContent(sb.String())
	// Auto-scroll to bottom to see newest if they are appended?
	// If the API returns newest first (desc), we might want to print them normally.
	// Let's assume standard log view.
}

func (m model) View() string {
	if !m.ready {
		return fmt.Sprintf("\n%s Initializing...", m.spinner.View())
	}

	// Top Pane: Identities / Usage
	// Since we don't have usage data yet, just list identities
	var identityList strings.Builder
	identityList.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("Active Identities") + "\n\n")

	if len(m.identities) == 0 {
		identityList.WriteString(subtleStyle.Render("No identities registered."))
	} else {
		// Sort identities for stable display
		ids := make([]string, 0, len(m.identities))
		for id := range m.identities {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		for _, id := range ids {
			ident := m.identities[id]
			identityList.WriteString(fmt.Sprintf("• %s (%s)\n", ident.IdentityID, ident.Kind))
		}
	}

	topPane := paneStyle.Render(identityList.String())

	// Bottom Pane: Event Stream
	header := headerStyle.Render(fmt.Sprintf("%s Activity Stream", m.spinner.View()))
	bottomPane := m.viewport.View()

	// Status Footer
	var status string
	if m.err != nil {
		status = errorStyle.Render(fmt.Sprintf("Offline: %v", m.err))
	} else {
		status = okStyle.Render(fmt.Sprintf("Online • %d Events • %d Identities", len(m.events), len(m.identities)))
	}
	footer := subtleStyle.Render(fmt.Sprintf("\n%s\nPress q to quit", status))

	return lipgloss.JoinVertical(lipgloss.Left, topPane, header, bottomPane, footer)
}

// Commands

func fetchData() tea.Cmd {
	return func() tea.Msg {
		// Fetch Events
		events, err := getEvents()
		if err != nil {
			return dataMsg{err: err}
		}

		// Fetch Identities
		identities, err := getIdentities()
		if err != nil {
			return dataMsg{err: err}
		}

		return dataMsg{
			events:     events,
			identities: identities,
			err:        nil,
		}
	}
}

func getEvents() ([]Event, error) {
	c := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := c.Get(fmt.Sprintf("%s/v1/events?limit=%d", daemonURL, maxEvents))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("events status %d", resp.StatusCode)
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}
	return events, nil
}

func getIdentities() (map[string]Identity, error) {
	c := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := c.Get(fmt.Sprintf("%s/v1/identities", daemonURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identities status %d", resp.StatusCode)
	}

	var identities map[string]Identity
	if err := json.NewDecoder(resp.Body).Decode(&identities); err != nil {
		return nil, err
	}
	return identities, nil
}

func tick() tea.Cmd {
	return tea.Tick(pollRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
