@devcontainer_connect
Feature: silo devcontainer connect — Open an interactive shell in the devcontainer

  `silo devcontainer connect` opens an interactive shell session inside the
  running devcontainer. The devcontainer is named `<workspace-container-name>-dev`.
  It requires the devcontainer to exist and be running. It does not accept any arguments.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Connects to the running devcontainer

    Scenario: devcontainer connect opens an interactive shell
      Given the devcontainer "silo-abc12345-dev" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect`
      Then podman should run "exec" with "-ti" on "silo-abc12345-dev"
      And the command should be "/bin/sh"
      And the exit code should be 0

    Scenario: devcontainer connect prints a message before opening shell
      Given the devcontainer "silo-abc12345-dev" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect`
      Then the output should contain "Connecting to silo-abc12345-dev..."

  Rule: Requires devcontainer to be running

    Scenario: devcontainer connect fails if devcontainer is not running
      Given the devcontainer "silo-abc12345-dev" exists but is stopped
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect`
      Then the exit code should not be 0
      And the error should contain "not running"

    Scenario: devcontainer connect fails if devcontainer does not exist
      Given no devcontainer exists
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect`
      Then the exit code should not be 0
      And the error should contain "not found" or "does not exist"

  Rule: Exiting the shell leaves the devcontainer running

    Scenario: exiting the devcontainer connect shell does not stop the devcontainer
      Given the devcontainer "silo-abc12345-dev" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect`
      And the interactive session ends
      Then the devcontainer "silo-abc12345-dev" should still be running
      And podman should not run "stop" on "silo-abc12345-dev"

  Rule: Multiple sessions can be connected simultaneously

    Scenario: two parallel devcontainer connect calls create two independent shells
      Given the devcontainer "silo-abc12345-dev" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer connect` and `silo devcontainer connect` in parallel
      Then two independent shell sessions should be opened in "silo-abc12345-dev"

  Rule: Does not affect the workspace container

    Scenario: devcontainer connect does not check workspace container state
      Given the devcontainer "silo-abc12345-dev" is running
      And no workspace container exists
      When I run `silo devcontainer connect`
      Then podman should run "exec" with "-ti" on "silo-abc12345-dev"
      And the workspace container state should not cause failure