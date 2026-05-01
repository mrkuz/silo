@devcontainer_stop
Feature: silo devcontainer stop — Stop and remove the devcontainer

  `silo devcontainer stop` stops the devcontainer immediately (no grace period),
  then removes it. If the devcontainer is not running, it prints a message and still
  attempts to remove. If the devcontainer does not exist, it prints "not found".

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Running devcontainer is stopped and removed

    Scenario: stop terminates and removes the devcontainer
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer stop`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
      And podman should run "rm" with "-f" on "silo-abc12345-dev"
      And the output should contain "Stopping silo-abc12345-dev..."
      And the output should contain "Removing silo-abc12345-dev..."
      And the exit code should be 0

    Scenario: stop does not affect the workspace container
      Given the devcontainer "silo-abc12345-dev" is running
      And the workspace container "silo-abc12345" is running
      When I run `silo devcontainer stop`
      Then podman should run "stop" on "silo-abc12345-dev"
      And podman should run "rm" on "silo-abc12345-dev"
      But podman should not run "stop" on "silo-abc12345"
      And podman should not run "rm" on "silo-abc12345"

  Rule: Stopped devcontainer is removed

    Scenario: stopped devcontainer prints message and is removed
      Given the devcontainer "silo-abc12345-dev" exists but is stopped
      When I run `silo devcontainer stop`
      Then podman should run "rm" with "-f" on "silo-abc12345-dev"
      And the output should contain "silo-abc12345-dev is not running"
      And the output should contain "Removing silo-abc12345-dev..."
      And the exit code should be 0

  Rule: Non-existing devcontainer prints not found

    Scenario: absent devcontainer prints not found and exits 0
      Given no devcontainer exists
      When I run `silo devcontainer stop`
      Then the output should contain "silo-abc12345-dev not found"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: devcontainer stop fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo devcontainer stop`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
