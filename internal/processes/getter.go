package processes

import (
	"fmt"
	"portexec/internal/models"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Getter handles process information retrieval
type Getter struct{}

// NewGetter creates a new process getter
func NewGetter() *Getter {
	return &Getter{}
}

// GetProcessInfo retrieves detailed information about a process
func (g *Getter) GetProcessInfo(pid int32) (*models.ProcessInfo, error) {
	// Create process object
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to create process for PID %d: %w", pid, err)
	}

	// Get process name
	name, err := p.Name()
	if err != nil {
		name = "unknown"
	}

	// Get executable path
	exePath, err := p.Exe()
	if err != nil {
		exePath = ""
	}

	// Get parent PID
	parentPid, err := p.Ppid()
	if err != nil {
		parentPid = 0
	}

	// Get create time
	createTime, err := p.CreateTime()
	if err != nil {
		createTime = 0
	}

	// Calculate uptime
	var uptime time.Duration
	if createTime > 0 {
		uptime = time.Since(time.Unix(createTime/1000, 0))
	}

	return &models.ProcessInfo{
		PID:        pid,
		Name:       name,
		ExePath:    exePath,
		ParentPID:  parentPid,
		Uptime:     uptime,
		CreateTime: time.Unix(createTime/1000, 0),
	}, nil
}

// GetProcessName retrieves just the process name
func (g *Getter) GetProcessName(pid int32) (string, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return "", fmt.Errorf("failed to create process for PID %d: %w", pid, err)
	}

	name, err := p.Name()
	if err != nil {
		return "", fmt.Errorf("failed to get process name: %w", err)
	}

	return name, nil
}

// GetProcessPath retrieves the executable path of a process
func (g *Getter) GetProcessPath(pid int32) (string, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return "", fmt.Errorf("failed to create process for PID %d: %w", pid, err)
	}

	exePath, err := p.Exe()
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}

	return exePath, nil
}

// GetParentProcessInfo retrieves information about the parent process
func (g *Getter) GetParentProcessInfo(pid int32) (*models.ProcessInfo, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to create process for PID %d: %w", pid, err)
	}

	parentPid, err := p.Ppid()
	if err != nil {
		return nil, fmt.Errorf("failed to get parent PID: %w", err)
	}

	if parentPid == 0 {
		return nil, fmt.Errorf("process has no parent")
	}

	return g.GetProcessInfo(parentPid)
}

// IsProcessRunning checks if a process is still running
func (g *Getter) IsProcessRunning(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}

	running, _ := p.IsRunning()
	return running
}

// GetAllPIDs returns all running process IDs
func (g *Getter) GetAllPIDs() ([]int32, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to get PIDs: %w", err)
	}
	return pids, nil
}
