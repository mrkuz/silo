@stop
Feature: silo stop — Stop the workspace container

  `silo stop` stops the running workspace container immediately (no grace period).
  It is a no-op if the container is already stopped.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Stops the running container

    Scenario: stop terminates the container immediately
      Given the container "silo-abc12345" is running
      When I run `silo stop`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
      And the output should contain "Stopping silo-abc12345..."
      And the exit code should be 0

  Rule: No-op if container is already stopped

    Scenario: stopped container is not an error
      Given the container "silo-abc12345" is stopped
      When I run `silo stop`
      Then the output should contain "silo-abc12345 is not running"
      And no podman stop should be called
      And the exit code should be 0

  Rule: Stop does not remove container or image

    Scenario: stop only stops, it does not remove anything
      Given the container "silo-abc12345" is running
      When I run `silo stop`
      Then podman should run "stop" on "silo-abc12345"
      But podman should not run "rm" on "silo-abc12345"
      And podman should not run "rmi" on "silo-abc12345"
      And the container "silo-abc12345" should still exist

  Rule: Requires workspace to be initialized

    Scenario: stop fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo stop`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
