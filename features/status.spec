@status
Feature: silo status — Show workspace container status

  `silo status` prints whether the workspace container is currently running or stopped.
  It requires the workspace to have been initialized first.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Reports container running state

    Scenario: status shows Running when container is up
      Given the container "silo-abc12345" is running
      When I run `silo status`
      Then the output should contain "Running"
      And the exit code should be 0

    Scenario: status shows Stopped when container is not running
      Given the container "silo-abc12345" is stopped
      When I run `silo status`
      Then the output should contain "Stopped"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: status fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo status`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
