# PortExec

[![Release](https://img.shields.io/github/v/release/DFanso/PortExec)](https://github.com/DFanso/PortExec/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/DFanso/PortExec)](https://github.com/DFanso/PortExec)
[![Build Status](https://img.shields.io/github/actions/workflow/status/DFanso/PortExec/release.yml)](https://github.com/DFanso/PortExec/actions)
[![License](https://img.shields.io/github/license/DFanso/PortExec)](LICENSE)
[![Downloads](https://img.shields.io/github/downloads/DFanso/PortExec/total)](https://github.com/DFanso/PortExec/releases)

A keyboard-driven TUI application for Windows to inspect and kill processes bound to specific TCP/UDP ports.

## Features

- **Port Lookup**: View all active listening ports with process information
- **Process Kill**: Kill processes with confirmation dialog
- **Safety Layer**: Protects critical system processes (System, lsass, wininit, etc.)
- **Real-time Search**: Filter by port or process name instantly
- **Pagination**: Navigate through large lists easily
- **CLI Mode**: Use from command line for scripting
- **Admin Detection**: Warns when not running with administrator privileges

## Installation

### Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/DFanso/PortExec/releases)

### From Source

```bash
git clone https://github.com/DFanso/PortExec.git
cd PortExec
go build -o portexec.exe ./cmd/main.go
```

## Usage

### Interactive TUI Mode

```bash
portexec
```

### CLI Mode

```bash
# List all listening ports
portexec list

# List specific port
portexec list 3000

# Kill process on port
portexec kill 8080

# Check admin privileges
portexec check
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↑/↓ or j/k | Navigate list |
| PgUp/PgDn | Change page |
| Enter | View process details |
| k | Kill selected process |
| r | Refresh port list |
| / | Search/filter mode |
| h | Show help |
| q | Quit |

## Requirements

- Windows 10+
- Go 1.22+ (for building from source)
- Administrator privileges for killing protected processes


## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [gopsutil](https://github.com/shirou/gopsutil) - Process/Network info
- [Cobra](https://github.com/spf13/cobra) - CLI

## License

MIT License - see [LICENSE](LICENSE) for details.
