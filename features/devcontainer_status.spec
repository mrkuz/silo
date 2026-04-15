@devcontainer_status
Feature: silo devcontainer status — Show devcontainer status

  `silo devcontainer status` prints whether the devcontainer is running or stopped.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Reports devcontainer running state

    Scenario: status shows Running when devcontainer is up
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer status`
      Then the output should contain "Running"
      And the exit code should be 0

    Scenario: status shows Stopped when devcontainer is not running
      Given the devcontainer "silo-abc12345-dev" is stopped
      When I run `silo devcontainer status`
      Then the output should contain "Stopped"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: status fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo devcontainer status`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
