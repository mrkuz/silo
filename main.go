// silo is a CLI tool for creating per-directory developer sandbox containers.
package main

import (
	"fmt"
	"os"

	"github.com/mrkuz/silo/cmd"
)

const helpText = `silo - developer sandbox container

Usage:
  silo [--stop|--rm|--rmi] [-- args...]
  silo init [--podman|--no-podman] [--shared-volume|--no-shared-volume]
  silo build
  silo create [--dry-run]
  silo start
  silo volume setup
  silo connect
  silo exec <cmd> [args...]
  silo stop
  silo rm [-f|--force]
  silo rmi [-f|--force]
  silo status
  silo user init
  silo user build
  silo user rmi
  silo devcontainer
  silo devcontainer stop
  silo devcontainer rm [--force]
  silo devcontainer status
  silo help

Commands:
  (default)            Run lifecycle and connect to the silo container
  init                 Initialize user and workspace files
  build                Build the workspace image
  create               Create the container
  start                Start the container
  volume setup         Create directories on the shared volume
  connect              Connect to the silo container
  exec                 Run a command in the running container
  stop                 Stop the running container
  rm                   Remove the container
  rmi                  Remove the workspace image
  status               Print container status
  user init            Create user files
  user build           Build the user image
  user rmi             Remove the user image
  devcontainer         Generate .devcontainer.json
  devcontainer stop    Stop the devcontainer
  devcontainer rm      Remove the devcontainer
  devcontainer status  Print devcontainer status
  help                 Show this help

Default command flags:
  --stop  Stop the container when the session exits
  --rm    Stop and remove the container when the session exits
  --rmi   Stop, remove container, and remove image when the session exits
  -- ...  Pass remaining arguments to podman exec

Init flags:
  --podman             Enable Podman inside the container
  --no-podman          Disable Podman inside the container
  --shared-volume      Enable shared volume mount
  --no-shared-volume   Disable shared volume mount

Create flags:
  --dry-run  Print the podman create command without running it
  -- ...     Pass remaining arguments to podman create

Remove flags:
  -f, --force  Stop the container if it is running before removing

Remove image flags:
  -f, --force  Stop and remove the container before removing the image

Devcontainer rm flags:
  --force  Stop the container if it is running before removing`

var commands = map[string]func([]string) error{
	"init":                cmd.Init,
	"build":               cmd.WithoutArgs(cmd.Build),
	"create":              cmd.Create,
	"start":               cmd.WithoutArgs(cmd.Start),
	"volume setup":        cmd.WithoutArgs(cmd.VolumeSetup),
	"connect":             cmd.Connect,
	"exec":                cmd.Exec,
	"stop":                cmd.WithoutArgs(cmd.Stop),
	"rm":                  cmd.Remove,
	"rmi":                 cmd.RemoveImage,
	"status":              cmd.WithoutArgs(cmd.Status),
	"user init":           cmd.WithoutArgs(cmd.UserInit),
	"user build":          cmd.WithoutArgs(cmd.UserBuild),
	"user rmi":            cmd.WithoutArgs(cmd.UserRmi),
	"devcontainer":        cmd.WithoutArgs(cmd.DevcontainerGenerate),
	"devcontainer stop":   cmd.WithoutArgs(cmd.DevcontainerStop),
	"devcontainer rm":     cmd.DevcontainerRemove,
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
		cmd := os.Args[1]
		if run, ok := commands[cmd]; ok {
			if err := run(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		}
		switch cmd {
		case "help", "--help", "-h":
			fmt.Println(helpText)
			return
		default:
			if cmd[0] != '-' {
				fmt.Fprintf(os.Stderr, "silo: unknown command %q\n\n%s\n", cmd, helpText)
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
