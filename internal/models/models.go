package models

import (
	"fmt"
	"strings"
	"time"
)

// PortEntry represents a network connection on a specific port
type PortEntry struct {
	Protocol     string        // TCP or UDP
	LocalAddress string        // IP:Port format
	Port         uint32        // Port number
	PID          int32         // Process ID
	ProcessName  string        // Name of the process
	State        string        // Connection state (LISTENING, ESTABLISHED, etc.)
	ParentPID    int32         // Parent Process ID
	Uptime       time.Duration // Process uptime
	ExePath      string        // Full path to executable
	IsSystem     bool          // Whether this is a critical system process
}

// ProcessInfo holds detailed information about a process
type ProcessInfo struct {
	PID        int32
	Name       string
	ExePath    string
	ParentPID  int32
	Uptime     time.Duration
	CreateTime time.Time
}

// FilterCriteria defines criteria for filtering port entries
type FilterCriteria struct {
	Port        string // Port number or substring
	ProcessName string // Process name substring
	PID         string // PID substring
}

// IsEmpty returns true if no filter criteria are set
func (f FilterCriteria) IsEmpty() bool {
	return f.Port == "" && f.ProcessName == "" && f.PID == ""
}

// Matches checks if the given PortEntry matches the filter criteria
func (f FilterCriteria) Matches(entry PortEntry) bool {
	if !f.IsEmpty() {
		// Port filter
		if f.Port != "" {
			var portMatch bool
			fmtPort := fmt.Sprintf("%d", entry.Port)
			portMatch = f.Port == fmtPort || strings.Contains(fmtPort, f.Port)
			if !portMatch {
				return false
			}
		}
		// Process name filter
		if f.ProcessName != "" {
			if !strings.Contains(strings.ToLower(entry.ProcessName), strings.ToLower(f.ProcessName)) {
				return false
			}
		}
		// PID filter
		if f.PID != "" {
			var pidMatch bool
			fmtPID := fmt.Sprintf("%d", entry.PID)
			pidMatch = f.PID == fmtPID || strings.Contains(fmtPID, f.PID)
			if !pidMatch {
				return false
			}
		}
	}
	return true
}

// CriticalSystemProcesses contains names of processes that should not be killed lightly
var CriticalSystemProcesses = map[string]bool{
	"System":       true,
	"wininit":      true,
	"services":     true,
	"lsass":        true,
	"svchost":      true,
	"csrss":        true,
	"winlogon":     true,
	"smss":         true,
	"dwm":          true,
	"explorer":     false, // Not critical enough to block
	"Registry":     true,
	"smss.exe":     true,
	"csrss.exe":    true,
	"wininit.exe":  true,
	"services.exe": true,
	"lsass.exe":    true,
	"svchost.exe":  true,
	"winlogon.exe": true,
	"dwm.exe":      true,
}

// IsCriticalProcess checks if a process is considered critical system process
func IsCriticalProcess(name string) bool {
	// Check exact match first
	if CriticalSystemProcesses[name] {
		return true
	}
	// Check with .exe suffix
	if CriticalSystemProcesses[name+".exe"] {
		return true
	}
	// Check if it's a svchost variant (services running under svchost)
	if strings.HasPrefix(strings.ToLower(name), "svchost") {
		return true
	}
	return false
}
