package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/golang/glog"
	"github.com/urfave/cli"

	"gospy/pkg/proc"
	"gospy/pkg/term"
)

func validPC(pc string) error {
	if pc != "current" && pc != "start" && pc != "go" {
		return fmt.Errorf("Invalid pc type: %s", pc)
	}
	return nil
}

func main() {
	var pid int
	var refresh int
	var nonblocking bool
	var pcType string
	pidFlag := cli.IntFlag{
		Name:        "pid",
		Usage:       "target go process id to spy",
		Required:    true,
		Destination: &pid,
	}
	refreshFlag := cli.IntFlag{
		Name:        "refresh",
		Usage:       "refresh interval in seconds",
		Value:       2,
		Destination: &refresh,
	}
	nonblockingFlag := cli.BoolFlag{
		Name:        "non-blocking",
		Usage:       "Don't suspend target process",
		Destination: &nonblocking,
	}
	pcFlag := cli.StringFlag{
		Name:        "pc",
		Usage:       "The program counter type: current, go, start",
		Value:       "go",
		Destination: &pcType,
	}
	app := cli.NewApp()
	app.Name = "gospy"
	app.Usage = "Hmm..."
	app.Commands = []cli.Command{
		{
			Name:    "summary",
			Aliases: []string{"s"},
			Usage:   "Dump go process internal summary",
			Flags:   []cli.Flag{pidFlag, nonblockingFlag, pcFlag},
			Action: func(c *cli.Context) error {
				if err := validPC(pcType); err != nil {
					return err
				}
				p, err := proc.New(pid)
				if err != nil {
					return err
				}
				sum, err := p.Summary(!nonblocking)
				if err != nil {
					return err
				}
				fmt.Println(sum)
				gs, err := p.GetGoroutines(!nonblocking)
				if err != nil {
					return err
				}
				sort.Slice(gs, func(i, j int) bool {
					return gs[i].ID < gs[j].ID
				})
				fmt.Print("goroutines:\n\n")
				for _, g := range gs {
					status := g.Status.String()
					if g.Waiting() {
						status = "waiting for " + g.WaitReason.String()
					}
					fmt.Printf("%d - %s: %s \n", g.ID, status, g.GetLocation(pcType).String())
				}
				return nil
			},
		},
		{
			Name:    "dump",
			Aliases: []string{"d"},
			Usage:   "Dump go process stack trace",
			Flags:   []cli.Flag{pidFlag},
			Action: func(c *cli.Context) error {
				return nil
			},
		},
		{
			Name:    "top",
			Aliases: []string{"t"},
			Usage:   "top like interface of functions executing",
			Flags:   []cli.Flag{pidFlag, refreshFlag, nonblockingFlag, pcFlag},
			Action: func(c *cli.Context) error {
				if err := validPC(pcType); err != nil {
					return err
				}
				p, err := proc.New(pid)
				if err != nil {
					return err
				}

				t := term.NewTerm(p, refresh, nonblocking, pcType)
				if err := t.Display(); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:    "record",
			Aliases: []string{"r"},
			Usage:   "Record stack trace",
			Flags:   []cli.Flag{pidFlag},
			Action: func(c *cli.Context) error {
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		glog.Error(err)
		os.Exit(1)
	}
}
