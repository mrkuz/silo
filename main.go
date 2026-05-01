// silo is a CLI tool for creating per-directory developer containers.
package main

import (
	"fmt"
	"os"

	"github.com/mrkuz/silo/cmd"
)

var commands = map[string]func([]string) error{
	"init":                cmd.Init,
	"build":               cmd.WithoutArgs(cmd.Build),
	"create":              cmd.Create,
	"start":               cmd.WithoutArgs(cmd.Start),
	"volume setup":        cmd.WithoutArgs(cmd.VolumeSetup),
	"connect":             cmd.Connect,
	"exec":                cmd.Exec,
	"stop":                cmd.WithoutArgs(cmd.Stop),
	"rmi":                 cmd.RemoveImage,
	"status":              cmd.WithoutArgs(cmd.Status),
	"user init":           cmd.WithoutArgs(cmd.UserInit),
	"user build":          cmd.WithoutArgs(cmd.UserBuild),
	"user rmi":            cmd.WithoutArgs(cmd.UserRmi),
	"devcontainer":        cmd.WithoutArgs(cmd.DevcontainerGenerate),
	"devcontainer stop":   cmd.WithoutArgs(cmd.DevcontainerStop),
	"devcontainer status": cmd.WithoutArgs(cmd.DevcontainerStatus),
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
		default:
			if arg[0] != '-' {
				fmt.Fprintf(os.Stderr, "silo: unknown command %q\n\n%s\n", arg, cmd.HelpText)
				os.Exit(1)
			}
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
