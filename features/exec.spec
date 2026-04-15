@exec
Feature: silo exec — Run a command in the workspace container

  `silo exec` runs an arbitrary command inside the running workspace container.
  Unlike `silo connect`, it does not start the container or trigger the lifecycle
  chain — the container must already be running. It requires the workspace to have
  been initialized.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Executes a command in the running container

    Scenario: exec runs the given command in the container
      Given the container "silo-abc12345" is running
      When I run `silo exec -- echo hello`
      Then podman should run "exec" with "-ti" on "silo-abc12345" with args "echo" "hello"
      And the exit code should be 0

  Rule: Requires the container to be running

    Scenario: exec fails when container is not running
      Given the container "silo-abc12345" is stopped
      When I run `silo exec echo hello`
      Then the exit code should not be 0
      And the error should indicate "silo-abc12345 is not running"

    Scenario: exec does not start the container
      Given no container exists
      When I run `silo exec echo hello`
      Then the exit code should not be 0
      And no podman start should be called

  Rule: Requires workspace to be initialized

    Scenario: exec fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo exec echo hello`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
