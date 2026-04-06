package main

import (
	"fmt"
	"os"
)

const helpText = `silo - developer sandbox container

Usage:
  silo [--stop] [-- args...]
  silo init
  silo build [--base] [-f|--force]
  silo create [--nested] [--no-workspace] [--no-shared-volume] [-f|--force] [--dry-run] [-- args...]
  silo start [-f|--force]
  silo setup
  silo connect [--stop] [-- args...]
  silo exec <cmd> [args...]
  silo stop
  silo rm [-f|--force] [--image]
  silo status
  silo devcontainer
  silo devcontainer stop
  silo devcontainer rm [-f|--force]
  silo devcontainer status
  silo help

Commands:
  (default)            Alias for connect
  init                 Initialize workspace and user files
  build                Build the workspace image
  create               Create the container
  start                Start the container
  setup                Run post-start setup in the running container
  connect              Connect to the silo container
  exec                 Run a command in the running container
  stop                 Stop the running container
  rm                   Remove the container
  status               Print container status
  devcontainer         Generate .devcontainer.json
  devcontainer stop    Stop the devcontainer
  devcontainer rm      Remove the devcontainer
  devcontainer status  Print devcontainer status
  help                 Show this help

Connect flags:
  --stop   Stop the container when the session exits
  -- ...   Pass remaining arguments to podman exec

Build flags:
  --base       Build the base and workspace image
  -f, --force  Remove and rebuild the image if it already exists

Create flags:
  --nested            Enable nested Podman containers
  --no-workspace      Disable workspace volume mount
  --no-shared-volume  Disable shared volume mount
  -f, --force         Remove and recreate the container if it already exists
  --dry-run           Print the podman command without running it
  -- ...              Pass remaining arguments to podman

Start flags:
  -f, --force  Restart the container if it is already running

Remove flags:
  -f, --force  Stop the container if it is running before removing
  --image      Also remove the workspace image

Devcontainer rm flags:
  -f, --force  Stop the container if it is running before removing`

var commands = map[string]func([]string) error{
	"init":              withoutArgs(cmdInit),
	"build":             cmdBuild,
	"create":            cmdCreate,
	"start":             cmdStart,
	"setup":             withoutArgs(cmdSetup),
	"connect":           cmdConnect,
	"exec":              cmdExec,
	"stop":              withoutArgs(cmdStop),
	"rm":                cmdRemove,
	"status":            withoutArgs(cmdStatus),
	"devcontainer":      withoutArgs(cmdDevcontainerGenerate),
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
	if err := cmdConnect(os.Args[1:]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "silo:", err)
	os.Exit(1)
}
