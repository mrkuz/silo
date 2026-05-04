package cmd

import "fmt"

// HelpText is the full command reference for silo.
const HelpText = `silo - developer container

Usage:
  silo [--stop]
  silo init [--podman|--no-podman]
  silo build [-f|--force]
  silo start
  silo volume setup
  silo connect
  silo stop
  silo rm [-f|--force]
  silo status
  silo user init
  silo user build [-f|--force]
  silo user rm
  silo devcontainer
  silo devcontainer stop
  silo devcontainer status
  silo help

Commands:
  (default)            Run lifecycle and connect to the silo container
  init                 Initialize user and workspace files
  build                Build the workspace image
  start                Start the container
  volume setup         Create directories on the shared volume
  connect              Connect to the silo container
  stop                 Stop and remove the running container
  rm                   Remove the workspace image
  status               Print container status
  user init            Create user files
  user build           Build the user image
  user rm              Remove the user image
  devcontainer         Generate .devcontainer.json
  devcontainer stop    Stop and remove the devcontainer
  devcontainer status  Print devcontainer status
  help                 Show this help

Default command flags:
  --stop  Stop and remove the container when the session exits

Init flags:
  --podman             Enable Podman inside the container
  --no-podman          Disable Podman inside the container
  -f, --force          Overwrite existing workspace files

Build flags:
  -f, --force          Force rebuild image; aborts if container exists or is running

User build flags:
  -f, --force          Force rebuild user image

Remove image flags:
  -f, --force  Stop and remove the container before removing the image`

// Help prints the command reference to stdout.
func Help() {
	fmt.Println(HelpText)
}

// PrintHelp returns the help text (for testing).
func PrintHelp() string {
	return HelpText
}
