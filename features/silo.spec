@silo
Feature: silo (default invocation) — Run lifecycle and connect to the container

  The default `silo` invocation (no subcommand) runs the full lifecycle chain
  (init → build → start) if needed, then opens an interactive shell session
  inside the running container. After the session exits, the --stop flag
  controls container cleanup.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Connects to the running container

    Scenario: default silo connects to the container
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo`
      Then podman should run "exec" with "-ti" on "silo-abc12345"

    Scenario: without cleanup flags, container keeps running after session ends
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo`
      And the interactive session ends
      Then the container "silo-abc12345" should still be running
      And podman should not run "stop" on "silo-abc12345"
      And podman should not run "rm" on "silo-abc12345"
      And the command should be "$HOME/.nix-profile/bin/default-shell"
      And the output should contain "Connecting to silo-abc12345..."

  Rule: --stop stops and removes the container after the session exits

    Scenario: container is stopped and removed after shell exits
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo --stop`
      And the interactive session ends
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
      And podman should run "rm" with "-f" on "silo-abc12345"

    Scenario: --stop does not remove the image
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo --stop`
      And the interactive session ends
      Then podman should not run "rmi" on "silo-abc12345"

  Rule: Runs the full lifecycle chain if needed

    Scenario: stopped container triggers start
      Given the container "silo-abc12345" exists but is stopped
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo`
      Then podman should run "start" on "silo-abc12345"
      And podman should run "exec" with "-ti" on "silo-abc12345"
      And the output should not contain "already exists"
      And the output should not contain "Building"

    Scenario: missing container triggers full build-and-create chain
      Given no container exists
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo`
      Then the container "silo-abc12345" should be created
      And podman should run "start" on "silo-abc12345"
      And podman should run "exec" with "-ti" on "silo-abc12345"
      And the output should contain "Creating silo-abc12345..."
      And the output should contain "Starting silo-abc12345..."

    Scenario: fresh workspace triggers full lifecycle: init, user image build, workspace image build, create, volume setup, start, connect
      Given a clean workspace with no existing silo files
      And the user's silo config directory has all starter files
      And no user image exists
      And no workspace image exists
      And no container exists
      When I run `silo`
      Then workspace files should be created: ".silo/silo.toml" and ".silo/home.nix"
      And the user image "silo-alice" should be built
      And the workspace image "silo-abc12345" should be built
      And the container "silo-abc12345" should be created
      And the container "silo-abc12345" should be running
      And podman should run "exec" with "-ti" on "silo-abc12345"

    Scenario: missing user image triggers user image build first
      Given no user image exists
      And the workspace image "silo-abc12345" exists
      And no container exists
      When I run `silo`
      Then the user image "silo-alice" should be built
      And the workspace image "silo-abc12345" should be built
      And the container "silo-abc12345" should be created
      And podman should run "exec" with "-ti" on "silo-abc12345"

    Scenario: missing workspace image triggers workspace image build
      Given the user image "silo-alice" exists
      And no workspace image exists
      And no container exists
      When I run `silo`
      Then the workspace image "silo-abc12345" should be built
      And the container "silo-abc12345" should be created
      And podman should run "exec" with "-ti" on "silo-abc12345"

    Scenario: volume setup runs before container start when shared volume is configured
      Given the config has paths ["$HOME/.cache/uv/"]
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      And no container exists
      When I run `silo`
      Then shared volume directories should be created
      And the container "silo-abc12345" should be created
      And the container "silo-abc12345" should be running
      And volume setup should complete before the container starts
