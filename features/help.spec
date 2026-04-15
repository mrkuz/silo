@help
Feature: silo help — Show command reference

  `silo help` prints the full command reference. It can also be triggered with
  `--help` or `-h` flags on any command.

  Scenario: help prints the command reference
    When I run `silo help`
    Then the output should contain "silo - developer sandbox container"
    And the output should contain "Usage:"
    And the output should contain "silo init"
    And the output should contain "silo build"
    And the output should contain "silo connect"
    And the output should contain "silo help"
    And the exit code should be 0

  Scenario: --help flag on silo prints the command reference
    When I run `silo --help`
    Then the output should contain "silo - developer sandbox container"
    And the exit code should be 0

  Scenario: -h flag on silo prints the command reference
    When I run `silo -h`
    Then the output should contain "silo - developer sandbox container"
    And the exit code should be 0
