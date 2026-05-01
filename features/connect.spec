@connect
Feature: silo connect — Open an interactive shell in the workspace container

  `silo connect` opens an interactive shell session inside the running workspace
  container. It requires the container to exist and be running. It does not accept
  any arguments.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Connects to the running container

    Scenario: connect opens an interactive shell
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect`
      Then podman should run "exec" with "-ti" on "silo-abc12345"
      And the command should be "/bin/sh"
      And the exit code should be 0

    Scenario: connect prints a message before opening shell
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect`
      Then the output should contain "Connecting to silo-abc12345..."


  Rule: Requires container to be running


    Scenario: connect fails if container is not running
      Given the container "silo-abc12345" exists but is stopped
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect`
      Then the exit code should not be 0
      And the error should contain "not running"


    Scenario: connect fails if container does not exist
      Given no container exists
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect`
      Then the exit code should not be 0
      And the error should contain "not found" or "does not exist"

  Rule: Exiting the shell leaves the container running

    Scenario: exiting the connect shell does not stop the container
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect`
      And the interactive session ends
      Then the container "silo-abc12345" should still be running
      And podman should not run "stop" on "silo-abc12345"

  Rule: Multiple sessions can be connected simultaneously

    Scenario: two parallel connect calls create two independent shells in the same container
      Given the container "silo-abc12345" is running
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo connect` and `silo connect` in parallel
      Then two independent shell sessions should be opened in "silo-abc12345"

