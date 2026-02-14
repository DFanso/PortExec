package ports

import (
	"fmt"
	"net"
	"portexec/internal/models"
	"portexec/internal/processes"
	"time"

	gopsutilnet "github.com/shirou/gopsutil/v3/net"
) // Scanner handles network port scanning and connection enumeration
type Scanner struct {
	processGetter *processes.Getter
}

// NewScanner creates a new port scanner
func NewScanner() *Scanner {
	return &Scanner{
		processGetter: processes.NewGetter(),
	}
}

// GetListeningPorts returns all listening ports with process information
func (s *Scanner) GetListeningPorts() ([]models.PortEntry, error) {
	return s.GetConnections([]string{"listen", "established"})
}

// GetConnections returns connections filtered by states
func (s *Scanner) GetConnections(states []string) ([]models.PortEntry, error) {
	// Get all network connections
	conns, err := gopsutilnet.Connections("all")
	if err != nil {
		return nil, fmt.Errorf("failed to get network connections: %w", err)
	}

	// Create a map to deduplicate connections by PID
	pidCache := make(map[int32]*models.ProcessInfo)

	var entries []models.PortEntry

	for _, conn := range conns {
		// Skip connections with no PID
		if conn.Pid == 0 {
			continue
		}

		// Only include TCP and UDP (type 1 = TCP, type 2 = UDP)
		if conn.Type != 1 && conn.Type != 2 {
			continue
		}

		// Convert state
		state := s.formatState(conn.Status)

		// Filter by state if specified
		if len(states) > 0 {
			if !containsState(states, state) {
				continue
			}
		}

		// Get local address and port
		localAddr := conn.Laddr.IP
		if localAddr == "" {
			localAddr = "0.0.0.0"
		}
		localPort := conn.Laddr.Port

		// Get process info (cached)
		procInfo, err := s.getProcessInfo(conn.Pid, pidCache)
		if err != nil {
			// Skip if we can't get process info
			continue
		}

		entry := models.PortEntry{
			Protocol:     s.protocolString(conn.Type),
			LocalAddress: net.JoinHostPort(localAddr, fmt.Sprintf("%d", localPort)),
			Port:         uint32(localPort),
			PID:          conn.Pid,
			ProcessName:  procInfo.Name,
			State:        state,
			ParentPID:    procInfo.ParentPID,
			Uptime:       procInfo.Uptime,
			ExePath:      procInfo.ExePath,
			IsSystem:     models.IsCriticalProcess(procInfo.Name),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// getProcessInfo returns cached process info or fetches it
func (s *Scanner) getProcessInfo(pid int32, cache map[int32]*models.ProcessInfo) (*models.ProcessInfo, error) {
	if info, ok := cache[pid]; ok {
		return info, nil
	}

	info, err := s.processGetter.GetProcessInfo(pid)
	if err != nil {
		return nil, err
	}

	cache[pid] = info
	return info, nil
}

// formatState converts connection status to display format
func (s *Scanner) formatState(status string) string {
	switch status {
	case "LISTEN":
		return "LISTENING"
	case "ESTABLISHED":
		return "ESTABLISHED"
	case "TIME_WAIT":
		return "TIME_WAIT"
	case "CLOSE_WAIT":
		return "CLOSE_WAIT"
	case "SYN_SENT":
		return "SYN_SENT"
	case "SYN_RECV":
		return "SYN_RECV"
	case "FIN_WAIT1":
		return "FIN_WAIT1"
	case "FIN_WAIT2":
		return "FIN_WAIT2"
	case "LAST_ACK":
		return "LAST_ACK"
	case "CLOSING":
		return "CLOSING"
	case "CLOSED":
		return "CLOSED"
	case "IDLE":
		return "IDLE"
	case "BOUND":
		return "BOUND"
	default:
		return status
	}
}

// protocolString returns the protocol name
func (s *Scanner) protocolString(t gopsutilnet.ConnectionType) string {
	switch t {
	case gopsutilnet.TCP:
		return "TCP"
	case gopsutilnet.UDP:
		return "UDP"
	default:
		return "UNKNOWN"
	}
}

// containsState checks if state is in the list
func containsState(states []string, state string) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

// Refresh reloads the connection data
func (s *Scanner) Refresh() ([]models.PortEntry, error) {
	return s.GetConnections([]string{"listen", "established"})
}

// GetPortEntry finds a specific port entry by PID and port
func (s *Scanner) GetPortEntry(pid int32, port uint32) (*models.PortEntry, error) {
	entries, err := s.GetConnections(nil)
	if err != nil {
		return nil, err
	}

	for i := range entries {
		if entries[i].PID == pid && entries[i].Port == port {
			return &entries[i], nil
		}
	}

	return nil, fmt.Errorf("port entry not found")
}

// GetEntriesByPort returns all entries for a specific port
func (s *Scanner) GetEntriesByPort(port uint32) ([]models.PortEntry, error) {
	entries, err := s.GetConnections(nil)
	if err != nil {
		return nil, err
	}

	var result []models.PortEntry
	for i := range entries {
		if entries[i].Port == port {
			result = append(result, entries[i])
		}
	}

	return result, nil
}

// GetEntriesByPID returns all entries for a specific PID
func (s *Scanner) GetEntriesByPID(pid int32) ([]models.PortEntry, error) {
	entries, err := s.GetConnections(nil)
	if err != nil {
		return nil, err
	}

	var result []models.PortEntry
	for i := range entries {
		if entries[i].PID == pid {
			result = append(result, entries[i])
		}
	}

	return result, nil
}

// IsValidPort checks if the port number is valid
func IsValidPort(port string) bool {
	var p int
	_, err := fmt.Sscanf(port, "%d", &p)
	if err != nil {
		return false
	}
	return p >= 0 && p <= 65535
}

// ConvertUptime converts a create time to uptime duration
func ConvertUptime(createTime int64) time.Duration {
	if createTime == 0 {
		return 0
	}
	return time.Since(time.Unix(createTime, 0))
}
