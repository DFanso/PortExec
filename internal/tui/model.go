package tui

import (
	"fmt"
	"portexec/internal/killer"
	"portexec/internal/models"
	"portexec/internal/ports"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the TUI state
type Model struct {
	// Data
	entries  []models.PortEntry
	filtered []models.PortEntry
	selected int
	filter   models.FilterCriteria

	// UI State
	isLoading       bool
	showHelp        bool
	showKillConfirm bool
	showDetails     bool
	searchMode      bool
	searchQuery     string
	errMsg          string
	successMsg      string

	// Pagination
	page     int
	pageSize int

	// Admin status
	isAdmin bool

	// Services
	scanner *ports.Scanner
	kill    *killer.Killer

	// Refreshing
	mu         sync.RWMutex
	refreshing bool

	// For details view
	selectedEntry models.PortEntry
}

// InitialModel creates the initial TUI model
func InitialModel() *Model {
	scanner := ports.NewScanner()
	kill := killer.NewKiller()

	// Check if running as admin
	isAdmin := killer.IsElevated()

	return &Model{
		entries:         []models.PortEntry{},
		filtered:        []models.PortEntry{},
		selected:        0,
		filter:          models.FilterCriteria{},
		isLoading:       true,
		showHelp:        false,
		showKillConfirm: false,
		showDetails:     false,
		searchMode:      false,
		searchQuery:     "",
		errMsg:          "",
		successMsg:      "",
		page:            0,
		pageSize:        20,
		isAdmin:         isAdmin,
		scanner:         scanner,
		kill:            kill,
		refreshing:      false,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return m.refresh()
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case refreshResult:
		m.mu.Lock()
		m.entries = msg.entries
		m.applyFilter()
		m.isLoading = false
		m.refreshing = false
		m.mu.Unlock()
		return m, nil

	case refreshError:
		m.mu.Lock()
		m.errMsg = msg.err.Error()
		m.isLoading = false
		m.refreshing = false
		m.mu.Unlock()
		return m, nil

	case tea.WindowSizeMsg:
		// Handle window resize if needed
	}

	// Clear messages after delay
	if m.errMsg != "" || m.successMsg != "" {
		go func() {
			time.Sleep(3 * time.Second)
			// Note: We can't directly modify model from goroutine
			// This would need to be handled differently in production
		}()
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (*Model, tea.Cmd) {
	// If in search mode, handle search input
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	// If showing help, close on any key
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// If showing kill confirmation
	if m.showKillConfirm {
		return m.handleKillConfirm(msg)
	}

	// If showing details
	if m.showDetails {
		key := msg.String()
		if key == "esc" || key == "enter" {
			m.showDetails = false
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "k":
		m2, cmd := m.handleKill()
		return m2, cmd

	case "r":
		return m, m.refresh()

	case "/":
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case "h":
		m.showHelp = true
		return m, nil

	case "up", "w": // vi style - up arrow or w
		if m.selected > 0 {
			m.selected--
		}
		return m, nil

	case "down", "j": // vi style
		if m.selected < len(m.getCurrentPageEntries())-1 {
			m.selected++
		}
		return m, nil

	case "pgup":
		m.prevPage()
		return m, nil

	case "pgdown":
		m.nextPage()
		return m, nil

	case "enter":
		pageEntries := m.getCurrentPageEntries()
		if len(pageEntries) > 0 && m.selected < len(pageEntries) {
			m.selectedEntry = pageEntries[m.selected]
			m.showDetails = true
		}
		return m, nil

	case "esc":
		// Clear filter if in filter mode
		if !m.filter.IsEmpty() {
			m.filter = models.FilterCriteria{}
			m.applyFilter()
			m.selected = 0
		}
		return m, nil
	}

	return m, nil
}

// handleSearchInput handles input in search mode
func (m *Model) handleSearchInput(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		m.filter = models.FilterCriteria{}
		m.applyFilter()
		return m, nil

	case "enter":
		m.searchMode = false
		m.selected = 0
		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		m.applyFilterFromSearch()
		m.selected = 0
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.applyFilterFromSearch()
			m.selected = 0
		}
		return m, nil
	}
}

// applyFilterFromSearch applies the search query as a filter
func (m *Model) applyFilterFromSearch() {
	if m.searchQuery == "" {
		m.filter = models.FilterCriteria{}
	} else {
		// Try to parse as port number first
		if ports.IsValidPort(m.searchQuery) {
			m.filter = models.FilterCriteria{Port: m.searchQuery}
		} else {
			// Otherwise treat as process name
			m.filter = models.FilterCriteria{ProcessName: m.searchQuery}
		}
	}
	m.applyFilter()
}

// applyFilter applies the current filter to entries
func (m *Model) applyFilter() {
	if m.filter.IsEmpty() {
		m.filtered = m.entries
	} else {
		m.filtered = make([]models.PortEntry, 0)
		for _, e := range m.entries {
			if m.filter.Matches(e) {
				m.filtered = append(m.filtered, e)
			}
		}
	}
	m.page = 0
	m.selected = 0
}

// handleKill initiates the kill process
func (m *Model) handleKill() (*Model, tea.Cmd) {
	pageEntries := m.getCurrentPageEntries()
	if len(pageEntries) == 0 || m.selected >= len(pageEntries) {
		m.errMsg = "No process selected"
		return m, nil
	}

	entry := pageEntries[m.selected]

	if entry.IsSystem {
		m.showKillConfirm = true
		return m, nil
	}

	m.showKillConfirm = true
	return m, nil
}

// handleKillConfirm handles the kill confirmation dialog
func (m *Model) handleKillConfirm(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showKillConfirm = false
		pageEntries := m.getCurrentPageEntries()
		if m.selected < len(pageEntries) {
			entry := pageEntries[m.selected]

			result := m.kill.Kill(entry.PID)
			if result.Success {
				m.successMsg = result.Message
				return m, m.refresh()
			} else {
				m.errMsg = result.Message
			}
		}

	case "n", "N", "esc":
		m.showKillConfirm = false

	default:
		// Ignore other keys
	}

	return m, nil
}

// Pagination methods
func (m *Model) getCurrentPageEntries() []models.PortEntry {
	start := m.page * m.pageSize
	end := start + m.pageSize
	if start >= len(m.filtered) {
		return []models.PortEntry{}
	}
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	return m.filtered[start:end]
}

func (m *Model) totalPages() int {
	if len(m.filtered) == 0 {
		return 1
	}
	return (len(m.filtered) + m.pageSize - 1) / m.pageSize
}

func (m *Model) nextPage() {
	if m.page < m.totalPages()-1 {
		m.page++
		m.selected = 0
	}
}

func (m *Model) prevPage() {
	if m.page > 0 {
		m.page--
		m.selected = 0
	}
}

// refresh triggers a data refresh
func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		m.mu.Lock()
		if m.refreshing {
			m.mu.Unlock()
			return nil
		}
		m.refreshing = true
		m.mu.Unlock()

		entries, err := m.scanner.GetListeningPorts()
		if err != nil {
			return refreshError{err}
		}

		return refreshResult{entries}
	}
}

// View renders the TUI
func (m *Model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(headerStyle.Render("PortExec v1.0"))
	sb.WriteString("\n\n")

	// Admin warning
	if !m.isAdmin {
		sb.WriteString(warningStyle.Render("Running without admin privileges. Some processes cannot be terminated."))
		sb.WriteString("\n\n")
	}

	// Filter display
	if !m.filter.IsEmpty() {
		sb.WriteString(fmt.Sprintf("Filter: [%s]  ", m.filter.Port+m.filter.ProcessName+m.filter.PID))
	}

	// Pagination info
	if len(m.filtered) > 0 {
		sb.WriteString(fmt.Sprintf("Page %d/%d (%d total)\n\n", m.page+1, m.totalPages(), len(m.filtered)))
	} else {
		sb.WriteString("\n")
	}

	// Loading state
	if m.isLoading {
		sb.WriteString(loadingStyle.Render("Loading..."))
		return sb.String()
	}

	// Help overlay
	if m.showHelp {
		return m.renderHelp()
	}

	// Kill confirmation overlay
	if m.showKillConfirm {
		return m.renderKillConfirm()
	}

	// Details overlay
	if m.showDetails {
		return m.renderDetails()
	}

	// Search mode
	if m.searchMode {
		sb.WriteString(searchStyle.Render(fmt.Sprintf("Search: %s", m.searchQuery)))
		sb.WriteString(" (Esc to cancel)\n\n")
	}

	// Error message
	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render(m.errMsg))
		sb.WriteString("\n\n")
		m.errMsg = "" // Clear after displaying
	}

	// Success message
	if m.successMsg != "" {
		sb.WriteString(successStyle.Render(m.successMsg))
		sb.WriteString("\n\n")
		m.successMsg = "" // Clear after displaying
	}

	// Table header
	sb.WriteString(tableHeaderStyle.Render(
		fmt.Sprintf("%-6s %-6s %-6s %-20s %-12s",
			"PROTO", "PORT", "PID", "PROCESS", "STATE"),
	))
	sb.WriteString("\n")

	// Table rows
	pageEntries := m.getCurrentPageEntries()
	for i, entry := range pageEntries {
		row := m.renderRow(entry)
		if i == m.selected {
			sb.WriteString(selectedRowStyle.Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		sb.WriteString(emptyStyle.Render("No ports found"))
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("\n")
	sb.WriteString(footerStyle.Render(
		"[↑/↓] Navigate [PgUp/PgDn] Page [k] Kill [r] Refresh [/] Filter [h] Help [q] Quit",
	))

	return sb.String()
}

// renderRow renders a single table row
func (m *Model) renderRow(entry models.PortEntry) string {
	stateStyle := getStateStyle(entry.State)
	return fmt.Sprintf("%-6s %-6d %-6d %-20s %s",
		entry.Protocol,
		entry.Port,
		entry.PID,
		truncate(entry.ProcessName, 20),
		stateStyle.Render(entry.State),
	)
}

// renderHelp renders the help overlay
func (m *Model) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(helpTitleStyle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	helpItems := []struct {
		key  string
		desc string
	}{
		{"↑/↓ or j/k", "Navigate list"},
		{"PgUp/PgDn", "Change page"},
		{"Enter", "View process details"},
		{"k", "Kill selected process"},
		{"r", "Refresh port list"},
		{"/", "Search/filter mode"},
		{"h", "Show this help"},
		{"q", "Quit"},
		{"Esc", "Clear filter / Close dialog"},
	}

	for _, item := range helpItems {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render(item.key), item.desc))
	}

	sb.WriteString("\n")
	sb.WriteString(helpCloseStyle.Render("Press any key to close"))

	return helpOverlayStyle.Render(sb.String())
}

// renderKillConfirm renders the kill confirmation dialog
func (m *Model) renderKillConfirm() string {
	entry := m.filtered[m.selected]

	var sb strings.Builder

	if entry.IsSystem {
		sb.WriteString(criticalWarningStyle.Render("⚠ CRITICAL SYSTEM PROCESS ⚠"))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("You are about to kill: %s (PID: %d)\n", entry.ProcessName, entry.PID))
		sb.WriteString("This is a critical system process. Killing it may cause system instability.\n")
		sb.WriteString("\n")
		sb.WriteString(warningStyle.Render("Are you absolutely sure? This cannot be undone!"))
	} else {
		sb.WriteString(fmt.Sprintf("Kill process: %s (PID: %d)?\n", entry.ProcessName, entry.PID))
	}

	sb.WriteString("\n")
	sb.WriteString(confirmStyle.Render("[Y] Yes, kill it  [N] Cancel"))

	return confirmOverlayStyle.Render(sb.String())
}

// renderDetails renders the process details view
func (m *Model) renderDetails() string {
	entry := m.selectedEntry

	var sb strings.Builder

	sb.WriteString(detailsTitleStyle.Render("Process Details"))
	sb.WriteString("\n\n")

	details := []struct {
		label string
		value string
	}{
		{"Protocol", entry.Protocol},
		{"Local Address", entry.LocalAddress},
		{"Port", fmt.Sprintf("%d", entry.Port)},
		{"PID", fmt.Sprintf("%d", entry.PID)},
		{"Process Name", entry.ProcessName},
		{"State", entry.State},
		{"Parent PID", fmt.Sprintf("%d", entry.ParentPID)},
		{"Uptime", formatUptime(entry.Uptime)},
		{"Executable", entry.ExePath},
	}

	for _, d := range details {
		sb.WriteString(fmt.Sprintf("%s: %s\n", detailLabelStyle.Render(d.label), d.value))
	}

	sb.WriteString("\n")
	sb.WriteString(detailsCloseStyle.Render("Press Enter or Esc to close"))

	return detailsOverlayStyle.Render(sb.String())
}

// truncate truncates a string to the given length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatUptime formats the uptime duration
func formatUptime(d time.Duration) string {
	if d == 0 {
		return "unknown"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 60
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// getStateStyle returns the appropriate style for a connection state
func getStateStyle(state string) lipgloss.Style {
	switch state {
	case "LISTENING":
		return stateListeningStyle
	case "ESTABLISHED":
		return stateEstablishedStyle
	case "TIME_WAIT", "CLOSE_WAIT":
		return stateWaitingStyle
	default:
		return stateDefaultStyle
	}
}

// Messages for refresh operations
type refreshResult struct {
	entries []models.PortEntry
}

type refreshError struct {
	err error
}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 0, 1, 0)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0, 0, 0)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("249")).
				Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("57")).
				Bold(false)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235"))

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Italic(true)

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

	// State styles
	stateListeningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")) // Green

	stateEstablishedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("75")) // Blue

	stateWaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")) // Gray

	stateDefaultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("white"))

	// Help overlay
	helpOverlayStyle = lipgloss.NewStyle().
				Width(50).
				Height(20).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("75")).
				Background(lipgloss.Color("235"))

	helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

	helpCloseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Confirm overlay
	confirmOverlayStyle = lipgloss.NewStyle().
				Width(60).
				Height(15).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("226")).
				Background(lipgloss.Color("235"))

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	criticalWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true).
				Background(lipgloss.Color("235"))

	// Details overlay
	detailsOverlayStyle = lipgloss.NewStyle().
				Width(50).
				Height(20).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("86")).
				Background(lipgloss.Color("235"))

	detailsTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("75")).
				Bold(true)

	detailsCloseStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)
)
