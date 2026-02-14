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

## Run Anywhere in Terminal

### Option 1: Add to PATH

1. Create a folder for your executables (e.g., `C:\Bin`)
2. Move `portexec.exe` to that folder
3. Add to PATH:
   - Press `Win + R`, type `sysdm.cpl`, press Enter
   - Go to **Advanced** > **Environment Variables**
   - Under **User variables**, find **Path** and click **Edit**
   - Click **New** and add `C:\Bin`
   - Click **OK** on all dialogs
4. Open a new terminal and run `portexec`

### Option 2: Windows Terminal Profile (Recommended)

1. Download the release and extract to a folder (e.g., `C:\Program Files\PortExec`)
2. Right-click the `.exe` file > **Create shortcut**
3. Right-click the shortcut > **Properties**
4. In the **Shortcut** tab, click **Advanced**
5. Check **Run as administrator**
6. Move the shortcut to your Desktop or Start Menu

### Option 3: Winget

```powershell
winget install --id DFanso.PortExec --source winget
```

## Running with Admin Privileges

PortExec requires administrator privileges to kill protected system processes.

### Method 1: Run as Administrator

1. Right-click on `portexec.exe`
2. Select **Run as administrator**

### Method 2: From PowerShell/CMD

```powershell
Start-Process -FilePath "portexec.exe" -Verb RunAs
```

### Method 3: From Windows Terminal

1. Click the dropdown arrow in the terminal tab
2. Select **Run as administrator**

### Verification

Run `portexec check` to verify admin status:
- If admin: Shows "Running with administrator privileges"
- If not admin: Shows "NOT running with administrator privileges" and some processes cannot be killed

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
