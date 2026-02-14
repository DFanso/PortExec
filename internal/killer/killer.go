package killer

import (
	"fmt"
	"os"
	"portexec/internal/models"

	"github.com/shirou/gopsutil/v3/process"
)

// Result represents the result of a kill operation
type Result struct {
	Success bool
	Message string
	Error   error
}

// Killer handles safe process termination
type Killer struct{}

// NewKiller creates a new killer instance
func NewKiller() *Killer {
	return &Killer{}
}

// Kill attempts to terminate a process by PID
func (k *Killer) Kill(pid int32) Result {
	// First check if process is critical
	procName, err := k.getProcessName(pid)
	if err != nil {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Failed to get process name: %v", err),
			Error:   err,
		}
	}

	if models.IsCriticalProcess(procName) {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Refusing to kill critical system process: %s", procName),
			Error:   fmt.Errorf("critical system process"),
		}
	}

	// Try to get the process
	p, err := process.NewProcess(pid)
	if err != nil {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Process %d not found (may have already terminated)", pid),
			Error:   err,
		}
	}

	// Try Kill first (force kill)
	err = p.Kill()
	if err != nil {
		// If Kill fails, try Terminate
		err = p.Terminate()
		if err != nil {
			return k.handleError(err, pid, procName)
		}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("Successfully killed process %d (%s)", pid, procName),
		Error:   nil,
	}
}

// KillWithName kills a process by PID with additional safety checks
func (k *Killer) KillWithName(pid int32, expectedName string) Result {
	// Verify process name matches
	procName, err := k.getProcessName(pid)
	if err != nil {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Failed to verify process: %v", err),
			Error:   err,
		}
	}

	if procName != expectedName {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Process name mismatch: expected %s, got %s", expectedName, procName),
			Error:   fmt.Errorf("process name mismatch"),
		}
	}

	return k.Kill(pid)
}

// ForceKill forcefully terminates a process without safety checks
func (k *Killer) ForceKill(pid int32) Result {
	p, err := process.NewProcess(pid)
	if err != nil {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Process %d not found", pid),
			Error:   err,
		}
	}

	err = p.Kill()
	if err != nil {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Failed to kill process: %v", err),
			Error:   err,
		}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("Force killed process %d", pid),
		Error:   nil,
	}
}

// getProcessName retrieves the name of a process
func (k *Killer) getProcessName(pid int32) (string, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return "", err
	}

	name, err := p.Name()
	if err != nil {
		return "", err
	}

	return name, nil
}

// handleError provides detailed error messages
func (k *Killer) handleError(err error, pid int32, procName string) Result {
	errStr := err.Error()

	// Check for common error types
	if contains(errStr, "access is denied") || contains(errStr, "Access is denied") {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Access denied. Run as Administrator to kill process %d (%s)", pid, procName),
			Error:   err,
		}
	}

	if contains(errStr, "operation not permitted") {
		return Result{
			Success: false,
			Message: fmt.Sprintf("Operation not permitted. Run as Administrator to kill process %d (%s)", pid, procName),
			Error:   err,
		}
	}

	return Result{
		Success: false,
		Message: fmt.Sprintf("Failed to kill process %d (%s): %v", pid, procName, err),
		Error:   err,
	}
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsElevated checks if the current process has admin privileges
func IsElevated() bool {
	// On Windows, we check if the process has SE_DEBUG_PRIVILEGE
	// This is a simplified check - in production you might want to use win32 API
	return isAdmin()
}

// isAdmin checks if running with administrator privileges
func isAdmin() bool {
	// Try to open a system process - this will fail if not admin
	// On Windows, we can check the process token
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		// Not a perfect check, but commonly used
		// A proper implementation would use Windows API
		return false
	}
	return true
}

// CheckKillAccess checks if we have permission to kill a process
func (k *Killer) CheckKillAccess(pid int32) (bool, string) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false, "Process not found"
	}

	// Try to get process info - this often fails without permissions
	_, err = p.Name()
	if err != nil {
		if contains(err.Error(), "access") || contains(err.Error(), "Access") {
			return false, "Access denied - requires administrator privileges"
		}
		return false, err.Error()
	}

	return true, "OK"
}
