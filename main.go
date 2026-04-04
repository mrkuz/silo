package main

import (
	"fmt"
	"os"
)

const helpText = `silo - developer sandbox container

Usage:
  silo [--stop] [-- args...]
  silo init
  silo build [--base] [--force]
  silo create [--nested] [--no-workspace] [--no-shared-volume] [--force] [--dry-run] [-- args...]
  silo start [--force]
  silo setup
  silo connect [--stop] [-- args...]
  silo exec <cmd> [args...]
  silo stop
  silo rm [--image]
  silo status
  silo devcontainer
  silo help

Commands:
  (default)     Alias for connect
  init          Initialize workspace and global config files
  build         Build the workspace image
  create        Create the container
  start         Start the container
  setup         Run post-start setup in the running container
  connect       Connect to the silo container
  exec          Run a command in the running container
  stop          Stop the running container
  rm            Remove the container
  status        Print container status
  devcontainer  Generate .devcontainer.json
  help          Show this help

Connect flags:
  --stop   Stop the container when the session exits
  -- ...   Pass remaining arguments to podman exec

Build flags:
  --base   Build the base and workspace image
  --force  Remove and rebuild the image if it already exists

Create flags:
  --nested            Enable nested Podman containers
  --no-workspace      Disable workspace volume mount
  --no-shared-volume  Disable shared volume
  --force             Remove and recreate the container if it already exists
  --dry-run           Print the podman command without running it
  -- ...              Pass remaining arguments to podman

Start flags:
  --force  Restart the container if it is already running

Remove flags:
  --image  Also remove the workspace image`

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "init":
			if err := cmdInit(); err != nil {
				fatal(err)
			}
			return
		case "build":
			if err := cmdBuild(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "create":
			if err := cmdCreate(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "start":
			if err := cmdStart(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "setup":
			if err := cmdSetup(); err != nil {
				fatal(err)
			}
			return
		case "connect":
			if err := cmdConnect(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "exec":
			if err := cmdExec(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "stop":
			if err := cmdStop(); err != nil {
				fatal(err)
			}
			return
		case "rm":
			if err := cmdRemove(os.Args[2:]); err != nil {
				fatal(err)
			}
			return
		case "status":
			if err := cmdStatus(); err != nil {
				fatal(err)
			}
			return
		case "devcontainer":
			if err := cmdDevcontainer(); err != nil {
				fatal(err)
			}
			return
		case "help", "--help", "-h":
			fmt.Println(helpText)
			return
		default:
			if os.Args[1][0] != '-' {
				fmt.Fprintf(os.Stderr, "silo: unknown command %q\n\n%s\n", os.Args[1], helpText)
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
