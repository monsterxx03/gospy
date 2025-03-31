package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/monsterxx03/gospy/pkg/api"
	"github.com/monsterxx03/gospy/pkg/proc"
	"github.com/monsterxx03/gospy/pkg/termui"
)

func main() {
	app := &cli.App{
		Name:  "gospy",
		Usage: "Process monitoring tool for Go applications",
		Commands: []*cli.Command{
			{
				Name:    "summary",
				Aliases: []string{"s"},
				Usage:   "Get process summary information",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "pid",
						Aliases:  []string{"p"},
						Usage:    "Target process ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "bin",
						Aliases: []string{"b"},
						Usage:   "Path to binary file (optional)",
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output in JSON format",
					},
				},
				Action: func(c *cli.Context) error {
					if os.Geteuid() != 0 {
						return fmt.Errorf("must be run as root")
					}
					pid := c.Int("pid")
					binPath := c.String("bin")

					// Create memory reader
					memReader, err := proc.NewProcessMemReader(pid, binPath)
					if err != nil {
						return fmt.Errorf("failed to create memory reader: %w", err)
					}
					defer memReader.Close()

					// Get Go version
					// Get runtime info
					rt, err := memReader.RuntimeInfo()
					if err != nil {
						return fmt.Errorf("failed to get runtime info: %w (is this a Go program?)", err)
					}

					// Get goroutines
					goroutines, err := memReader.Goroutines()
					if err != nil {
						return fmt.Errorf("failed to get goroutines: %w", err)
					}

					// Output format
					jsonOutput := c.Bool("json")
					if jsonOutput {
						type Summary struct {
							PID        int      `json:"pid"`
							GoVersion  string   `json:"go_version"`
							Goroutines []proc.G `json:"goroutines"`
						}
						summary := Summary{
							PID:        pid,
							GoVersion:  rt.GoVersion,
							Goroutines: goroutines,
						}
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(summary)
					}

					// Print summary
					fmt.Printf("\nProcess %d Summary:\n", pid)
					fmt.Printf("  Go Version: %s\n", rt.GoVersion)
					if !strings.HasPrefix(rt.GoVersion, "go") {
						fmt.Printf("  Warning: Unexpected version format: %q\n", rt.GoVersion)
					}

					fmt.Printf("\nGoroutines (%d):\n", len(goroutines))
					for i, g := range goroutines {
						status := g.Status
						if g.WaitReason != "" {
							status += fmt.Sprintf(" (%s)", g.WaitReason)
						}

						funcName := g.StartFuncName
						if funcName == "" {
							funcName = "unknown"
						}

						fmt.Printf("  [%4d] G%4d %-15s 0x%x [stack: 0x%x-0x%x] %s\n",
							i+1,
							g.Goid,
							status,
							g.Address,
							g.Stack.Lo,
							g.Stack.Hi,
							funcName)
					}

					return nil
				},
			},
			{
				Name:    "serve",
				Aliases: []string{"api"},
				Usage:   "Start API server to expose process information",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Port to listen on",
						Value:   8974,
					},
				},
				Action: func(c *cli.Context) error {
					if os.Geteuid() != 0 {
						return fmt.Errorf("must be run as root")
					}
					port := c.Int("port")
					apiServer := api.NewServer(port)
					fmt.Printf("Starting API server on port %d\n", port)
					fmt.Printf("Endpoints:\n")
					fmt.Printf("  GET /runtime?pid=<PID>     - Get runtime info\n")
					fmt.Printf("  GET /goroutines?pid=<PID> - Get goroutines list\n")
					fmt.Printf("  GET /memstats?pid=<PID>   - Get memory stats\n")
					return apiServer.Start()
				},
			},
			{
				Name:    "top",
				Aliases: []string{"t"},
				Usage:   "Monitor goroutines in a top-like interface",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "pid",
						Aliases:  []string{"p"},
						Usage:    "Target process ID",
						Required: true,
					},
					&cli.IntFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Usage:   "Refresh interval in seconds",
						Value:   2,
					},
					&cli.BoolFlag{
						Name:    "debug",
						Usage:   "Enable debug mode (wait for dlv attach)",
						Value:   false,
					},
				},
				Action: func(c *cli.Context) error {
					if os.Geteuid() != 0 {
						return fmt.Errorf("must be run as root")
					}
					pid := c.Int("pid")
					interval := c.Int("interval")
					if interval <= 0 {
						interval = 2
					}

					// Create memory reader
					memReader, err := proc.NewProcessMemReader(pid, "")
					if err != nil {
						return fmt.Errorf("failed to create memory reader: %w", err)
					}
					defer memReader.Close()

					// Wait for debugger if debug flag is set
					if c.Bool("debug") {
						fmt.Printf("Waiting for dlv to attach to PID %d...\n", os.Getpid())
						select {} // Block forever until debugger attaches
					}

					// Create and run top UI
					topUI := termui.NewTopUI(pid, interval, memReader)
					if err := topUI.Run(); err != nil {
						return fmt.Errorf("top UI error: %w", err)
					}

					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			fmt.Println("Welcome to procmon! Use 'summary --pid' to get process info")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
