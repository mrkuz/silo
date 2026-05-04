package cmd

import "fmt"

// HelpText is the full command reference for silo.
const HelpText = `Usage:
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
  silo devcontainer connect
  silo devcontainer stop
  silo devcontainer status
  silo help

Commands:
  (default)             Run lifecycle and connect to the silo container
    --stop                Stop and remove the container when the session exits
  init                  Initialize user and workspace files
    --podman              Enable Podman inside the container
    --no-podman           Disable Podman inside the container
  build                 Build the workspace image
    -f, --force           Force rebuild image; aborts if container exists or is running
  start                 Start the container
  volume setup          Create directories on the shared volume
  connect               Connect to the silo container
  stop                  Stop and remove the running container
  rm                    Remove the workspace image
    -f, --force           Stop and remove the container before removing the image
  status                Print container status
  user init             Create user files
  user build            Build the user image
    -f, --force           Force rebuild user image
  user rm               Remove the user image
  devcontainer          Generate .devcontainer.json
    -f, --force           Overwrite existing .devcontainer.json
  devcontainer connect  Connect to the devcontainer
  devcontainer stop     Stop and remove the devcontainer
  devcontainer status   Print devcontainer status
  help                  Show this help
`

// Help prints the command reference to stdout.
func Help() {
	fmt.Print(HelpText)
}

// PrintHelp returns the help text (for testing).
func PrintHelp() string {
	return HelpText
}
