package main

import (
	"fmt"
	"os"
	"portexec/internal/killer"
	"portexec/internal/models"
	"portexec/internal/ports"
	"portexec/internal/tui"
	"portexec/internal/version"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	portFlag     string
	killFlag     bool
	listenFlag   bool
	showPathFlag bool
)

func main() {
	// Parse command line flags
	if len(os.Args) > 1 {
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			printUsage()
			os.Exit(0)
		}

		// Check for CLI mode commands
		cliCommands := map[string]bool{
			"list":  true,
			"kill":  true,
			"check": true,
		}

		if cliCommands[os.Args[1]] {
			runCLI()
			return
		}
	}

	// Run TUI mode
	runTUI()
}

func runTUI() {
	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting application: %v\n", err)
		os.Exit(1)
	}
}

func runCLI() {
	rootCmd := &cobra.Command{
		Use:   "portexec",
		Short: "PortExec - Port process management tool",
		Long:  `A command-line tool to inspect and kill processes bound to specific TCP/UDP ports on Windows.`,
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list [port]",
		Short: "List processes on ports",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port := ""
			if len(args) > 0 {
				port = args[0]
			}
			return listPorts(port, listenFlag)
		},
	}
	listCmd.Flags().BoolVarP(&listenFlag, "listen", "l", false, "Show only listening ports")
	rootCmd.AddCommand(listCmd)

	// Kill command
	killCmd := &cobra.Command{
		Use:   "kill <port>",
		Short: "Kill process on a port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return killByPort(args[0])
		},
	}
	rootCmd.AddCommand(killCmd)

	// Check admin command
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check if running as administrator",
		Run: func(cmd *cobra.Command, args []string) {
			if killer.IsElevated() {
				fmt.Println("Running with administrator privileges")
				os.Exit(0)
			} else {
				fmt.Println("NOT running with administrator privileges")
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(checkCmd)

	// Flags
	rootCmd.PersistentFlags().BoolVarP(&showPathFlag, "path", "p", false, "Show executable path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func listPorts(port string, listenOnly bool) error {
	scanner := ports.NewScanner()

	var states []string
	if listenOnly {
		states = []string{"LISTENING"}
	}

	entries, err := scanner.GetConnections(states)
	if err != nil {
		return fmt.Errorf("failed to get connections: %w", err)
	}

	// Filter by port if specified
	if port != "" {
		var filtered []models.PortEntry
		for _, e := range entries {
			if fmt.Sprintf("%d", e.Port) == port {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Print header
	fmt.Printf("%-6s %-6s %-6s %-20s %-12s\n", "PROTO", "PORT", "PID", "PROCESS", "STATE")
	fmt.Println(strings.Repeat("-", 60))

	// Print entries
	for _, e := range entries {
		fmt.Printf("%-6s %-6d %-6d %-20s %-12s\n",
			e.Protocol, e.Port, e.PID, e.ProcessName, e.State)
		if showPathFlag && e.ExePath != "" {
			fmt.Printf("    Path: %s\n", e.ExePath)
		}
	}

	fmt.Printf("\nTotal: %d entries\n", len(entries))
	return nil
}

func killByPort(port string) error {
	scanner := ports.NewScanner()
	k := killer.NewKiller()

	// Get entries for the port
	entries, err := scanner.GetEntriesByPort(parsePort(port))
	if err != nil {
		return fmt.Errorf("failed to get port entries: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no process found on port %s", port)
	}

	// Get unique PIDs
	pidMap := make(map[int32]models.PortEntry)
	for _, e := range entries {
		pidMap[e.PID] = e
	}

	// Kill each unique PID
	fmt.Printf("Found %d unique process(es) on port %s:\n\n", len(pidMap), port)

	hasError := false
	for pid, entry := range pidMap {
		fmt.Printf("Killing %s (PID: %d)... ", entry.ProcessName, pid)

		result := k.Kill(pid)
		if result.Success {
			fmt.Println("OK")
		} else {
			fmt.Printf("FAILED: %s\n", result.Message)
			hasError = true
		}
	}

	if hasError {
		os.Exit(1)
	}

	return nil
}

func parsePort(port string) uint32 {
	var p uint32
	fmt.Sscanf(port, "%d", &p)
	return p
}

func printUsage() {
	fmt.Printf("PortExec v%s - Port Process Management Tool\n\n", version.Version)
	fmt.Print(`Usage:
  portexec                    Start interactive TUI
  portexec list [port]       List processes on ports
  portexec kill <port>       Kill process on port
  portexec check             Check admin privileges

Options:
  -l, --listen    Show only listening ports
  -p, --path      Show executable path

Keyboard Shortcuts (TUI mode):
  ↑/↓ or j/k    Navigate list
  PgUp/PgDn      Change page
  Enter          View process details
  k              Kill selected process
  r              Refresh port list
  /              Search/filter mode
  h              Show help
  q              Quit

Examples:
  portexec              # Start interactive TUI
  portexec list         # List all listening ports
  portexec list 3000    # List processes on port 3000
  portexec kill 8080    # Kill process on port 8080
`)
}
