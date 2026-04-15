@devcontainer_rm
Feature: silo devcontainer rm — Remove the devcontainer

  `silo devcontainer rm` removes the devcontainer. It refuses to remove a running
  devcontainer unless `--force` is given. It does not affect the workspace container.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Removes a stopped devcontainer

    Scenario: rm removes stopped devcontainer
      Given the devcontainer "silo-abc12345-dev" exists but is stopped
      When I run `silo devcontainer rm`
      Then podman should run "rm" with "-f" on "silo-abc12345-dev"
      And the output should contain "Removing silo-abc12345-dev..."
      And the exit code should be 0

  Rule: Refuses to remove a running devcontainer without --force

    Scenario: rm without --force returns an error
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer rm`
      Then the exit code should not be 0
      And the error should indicate "silo-abc12345-dev is running"
      And the devcontainer "silo-abc12345-dev" should still exist

  Rule: --force stops and removes a running devcontainer

    Scenario: --force stops running devcontainer before removing
      Given the devcontainer "silo-abc12345-dev" is running
      When I run `silo devcontainer rm --force`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
      And podman should run "rm" with "-f" on "silo-abc12345-dev"
      And the exit code should be 0

  Rule: Devcontainer not found is a no-op

    Scenario: missing devcontainer prints not found and exits 0
      Given no devcontainer exists
      When I run `silo devcontainer rm`
      Then the output should contain "silo-abc12345-dev not found"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: devcontainer rm fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo devcontainer rm`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing

  Rule: Does not stop or remove the workspace container or image

    Scenario: rm only removes the devcontainer, not the workspace container
      Given the devcontainer "silo-abc12345-dev" exists but is stopped
      And the workspace container "silo-abc12345" is running
      When I run `silo devcontainer rm`
      Then podman should run "rm" on "silo-abc12345-dev"
      But podman should not run "stop" on "silo-abc12345"
      And podman should not run "rm" on "silo-abc12345"
      And the workspace container "silo-abc12345" should still be running

    Scenario: rm only removes the devcontainer, not the workspace image
      Given the devcontainer "silo-abc12345-dev" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer rm`
      Then podman should run "rm" on "silo-abc12345-dev"
      But podman should not run "rmi" on "silo-abc12345"
      And the workspace image "silo-abc12345" should still exist
