package main

import (
	"os"

	"github.com/golang/glog"
	"github.com/urfave/cli"

	"gospy/pkg/proc"
	"gospy/pkg/term"
)

func main() {
	var pid int
	pidFlag := cli.IntFlag{
		Name:        "pid",
		Usage:       "target go process id to spy",
		Required:    true,
		Destination: &pid,
	}
	app := cli.NewApp()
	app.Name = "gospy"
	app.Usage = "Hmm..."
	app.Commands = []cli.Command{
		{
			Name:    "summary",
			Aliases: []string{"s"},
			Usage:   "Dump go process internal summary",
			Flags:   []cli.Flag{pidFlag},
			Action: func(c *cli.Context) error {
				p, err := proc.New(pid)
				if err != nil {
					return err
				}
				_, err = p.Summary()
				if err != nil {
					return err
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
			Flags:   []cli.Flag{pidFlag},
			Action: func(c *cli.Context) error {
				t := &term.Term{}

				p, err := proc.New(pid)
				if err != nil {
					return err
				}
				sum, err := p.Summary()
				if err != nil {
					return err
				}
				gs, err := p.GetGoroutines()
				if err != nil {
					return err
				}
				if err := t.Display([]string{sum.String()}, gs); err != nil {
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
