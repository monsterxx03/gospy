# Go Process Inspector

[![Go Report Card](https://goreportcard.com/badge/github.com/monsterxx03/gospy)](https://goreportcard.com/report/github.com/monsterxx03/gospy)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A tool for inspecting and analyzing running Go processes, including goroutine states, memory statistics, and binary information.

## Features

- View detailed goroutine information (status, scheduling info)
- Analyze process memory statistics
- Cross-platform support (Linux and macOS)
- Terminal UI for interactive inspection
- HTTP API for programmatic access

## Installation

```bash
go install github.com/monsterxx03/gospy@latest
```

## Usage

### CLI Interface

```bash
# Interactive terminal UI
sudo gospy top --pid <pid>

# HTTP API server
sudo gospy serve --port 8974

# Get process summary
sudo gospy summary --pid <pid>

# Get process summary in JSON format
sudo gospy summary --pid <pid> --json
```

#### Summary Command Options
- `--pid/-p` - Target process ID (required)
- `--bin/-b` - Path to binary file (optional)
- `--json/-j` - Output results in JSON format

### API Endpoints

- `GET /goroutines?pid=<pid>` - List all goroutines
- `GET /memstats?pid=<pid>` - Get memory statistics
- `GET /runtime?pid=<pid>` - Get runtime version info

### MCP Server (Machine Control Protocol)

The MCP server provides an SSE (Server-Sent Events) endpoint for real-time monitoring and control. To enable:

```bash
sudo gospy serve --enable-mcp --port 8974
```

Available MCP tools:
- `goroutines` - Dump goroutines for a process
  - Parameters:
    - `pid` (required) - Process ID to inspect

Example usage with MCP client:
```bash
# Connect to MCP SSE endpoint
curl -N http://localhost:8974/mcp/sse

# Sample tool call:
{"type":"call","tool":"goroutines","params":{"arguments":{"pid":1234}}}
```

### Terminal UI Controls

- `q` - Quit
- `r` - Refresh data
- `s` - Suspend/Resume top view
- `/` - Search/filter goroutines

### Terminal UI Screenshot

![Terminal UI Screenshot](screenshots/top.png)

## Building from Source

```bash
git clone https://github.com/monsterxx03/gospy.git
cd gospy
make
```

## Requirements

- Go 1.20+
- Linux or macOS (Apple Silicon only)
- Root privileges (required for memory access)

## Root Privileges

gospy requires root privileges to:
- Read process memory (/proc/<pid>/mem on Linux)
- Access Mach APIs on macOS

Run with sudo:
```bash
sudo gospy top --pid <pid>
```

For development/debugging, you may want to:
1. Build the binary first: `make`
2. Run with sudo: `sudo ./gospy [command]`

## Credits

Version 0.7.0 was completely rewritten from scratch with [aider](https://aider.chat), which wrote >90% of the code. Additional assistance from:
- [DeepSeek](https://deepseek.com) (R1 + V3 models) - AI coding assistant

Total AI compute cost: ~$2 USD

## License

MIT - See [LICENSE](LICENSE) file for details.
