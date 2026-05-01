@stop
Feature: silo stop — Stop and remove the workspace container

  `silo stop` stops the running workspace container immediately (no grace period),
  then removes the container. If the container is not running, it prints a message
  and still attempts to remove. If the container does not exist, it prints "not found".

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Running container is stopped and removed

    Scenario: stop terminates and removes the container
      Given the container "silo-abc12345" is running
      When I run `silo stop`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
      And podman should run "rm" with "-f" on "silo-abc12345"
      And the output should contain "Stopping silo-abc12345..."
      And the output should contain "Removing silo-abc12345..."
      And the exit code should be 0

  Rule: Stopped container is removed

    Scenario: stopped container prints message and is removed
      Given the container "silo-abc12345" exists but is stopped
      When I run `silo stop`
      Then podman should run "rm" with "-f" on "silo-abc12345"
      And the output should contain "silo-abc12345 is not running"
      And the output should contain "Removing silo-abc12345..."
      And the exit code should be 0

  Rule: Non-existing container prints not found

    Scenario: absent container prints not found and exits 0
      Given no container exists
      When I run `silo stop`
      Then the output should contain "silo-abc12345 not found"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: stop fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo stop`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
