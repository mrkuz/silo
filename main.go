// silo is a CLI tool for creating per-directory developer containers.
package main

import (
	"fmt"
	"os"

	"github.com/mrkuz/silo/cmd"
)

var commands = map[string]func([]string) error{
	"init":                 cmd.Init,
	"build":                cmd.Build,
	"start":                cmd.NoArgs(cmd.Start),
	"volume setup":         cmd.NoArgs(cmd.VolumeSetup),
	"connect":              cmd.NoArgs(cmd.Connect),
	"stop":                 cmd.NoArgs(cmd.Stop),
	"rm":                   cmd.NoArgs(cmd.Remove),
	"status":               cmd.NoArgs(cmd.Status),
	"user init":            cmd.UserInit,
	"user build":           cmd.UserBuild,
	"user rm":              cmd.NoArgs(cmd.UserRm),
	"devcontainer":         cmd.DevcontainerGenerate,
	"devcontainer connect": cmd.NoArgs(cmd.DevcontainerConnect),
	"devcontainer stop":    cmd.NoArgs(cmd.DevcontainerStop),
	"devcontainer status":  cmd.NoArgs(cmd.DevcontainerStatus),
}

func main() {
	if len(os.Args) >= 2 {
		// Try two-word command first (e.g. "devcontainer stop").
		if len(os.Args) >= 3 {
			compound := os.Args[1] + " " + os.Args[2]
			if run, ok := commands[compound]; ok {
				if err := run(os.Args[3:]); err != nil {
					fatal(err)
				}
				return
			}
		}
		arg := os.Args[1]
		if run, ok := commands[arg]; ok {
			if err := run(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		}
		switch arg {
		case "help", "--help", "-h":
			cmd.Help()
			return
		case "--stop":
			if err := cmd.Run(os.Args[1:]); err != nil {
				fatal(err)
			}
			return
		default:
			fatal(cmd.ErroneousCommand())
			return
		}
	}
	if err := cmd.Run(os.Args[1:]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "silo:", err)
	os.Exit(1)
}
