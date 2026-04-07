package main

import (
	"fmt"
	"os"
)

const helpText = `silo - developer sandbox container

Usage:
  silo [--stop|--rm|--rmi] [-- args...]
  silo init
  silo build [--user]
  silo create [--nested] [--shared-volume] [--dry-run] [-- args...]
  silo start
  silo setup
  silo connect
  silo exec <cmd> [args...]
  silo stop
  silo rm [-f|--force]
  silo rmi [-f|--force] [--user]
  silo status
  silo devcontainer
  silo devcontainer stop
  silo devcontainer rm [--force]
  silo devcontainer status
  silo help

Commands:
  (default)            Run lifecycle and connect to the silo container
  init                 Initialize workspace and user files
  build                Build the workspace image
  create               Create the container
  start                Start the container
  setup                Run post-start setup in the running container
  connect              Connect to the silo container
  exec                 Run a command in the running container
  stop                 Stop the running container
  rm                   Remove the container
  rmi                  Remove the workspace image
  status               Print container status
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

Build flags:
  --user  Build the user image

Create flags:
  --nested            Enable nested Podman containers
  --shared-volume     Enable shared volume mount
  --dry-run           Print the podman create command without running it
  -- ...              Pass remaining arguments to podman create

Remove flags:
  -f, --force  Stop the container if it is running before removing

Remove image flags:
  -f, --force  Stop and remove the container before removing the image
  --user       Remove the user image

Devcontainer rm flags:
  --force  Stop the container if it is running before removing`

var commands = map[string]func([]string) error{
	"init":                withoutArgs(cmdInit),
	"build":               cmdBuild,
	"create":              cmdCreate,
	"start":               withoutArgs(cmdStart),
	"setup":               withoutArgs(cmdSetup),
	"connect":             cmdConnect,
	"exec":                cmdExec,
	"stop":                withoutArgs(cmdStop),
	"rm":                  cmdRemove,
	"rmi":                 cmdRemoveImage,
	"status":              withoutArgs(cmdStatus),
	"devcontainer":        withoutArgs(cmdDevcontainerGenerate),
	"devcontainer stop":   withoutArgs(cmdDevcontainerStop),
	"devcontainer rm":     cmdDevcontainerRemove,
	"devcontainer status": withoutArgs(cmdDevcontainerStatus),
}

func withoutArgs(f func() error) func([]string) error {
	return func(_ []string) error { return f() }
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
	if err := cmdRun(os.Args[1:]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "silo:", err)
	os.Exit(1)
}
