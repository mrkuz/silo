@devcontainer_stop
Feature: silo devcontainer stop — Stop the devcontainer

  `silo devcontainer stop` stops the devcontainer. It is a no-op if the
  devcontainer is not running.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Stops the running devcontainer

    Scenario: stop stops the devcontainer
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer stop`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
      And the exit code should be 0

    Scenario: stop prints a message
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer stop`
      Then the output should contain "Stopping silo-abc12345-dev..."

    Scenario: stop does not stop the workspace container
      Given the devcontainer "silo-abc12345-dev" is running
      And the workspace container "silo-abc12345" is running
      When I run `silo devcontainer stop`
      Then podman should run "stop" on "silo-abc12345-dev"
      But podman should not run "stop" on "silo-abc12345"

  Rule: No-op if devcontainer is not running

    Scenario: stopped devcontainer is not an error
      Given the devcontainer "silo-abc12345-dev" is stopped
      When I run `silo devcontainer stop`
      Then the output should contain "silo-abc12345-dev is not running"
      And no podman stop should be called
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: devcontainer stop fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo devcontainer stop`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
