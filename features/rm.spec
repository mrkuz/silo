@rm
Feature: silo rm — Remove the workspace container

  `silo rm` removes the workspace container. It refuses to remove a running
  container unless `--force` is given. It does not remove the image.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Removes a stopped container

    Scenario: stopped container is removed without --force
      Given the container "silo-abc12345" exists but is stopped
      When I run `silo rm`
      Then podman should run "rm" with "-f" on "silo-abc12345"
      And the output should contain "Removing silo-abc12345..."
      And the exit code should be 0

  Rule: Refuses to remove a running container without --force

    Scenario: running container without --force returns an error
      Given the container "silo-abc12345" is running
      When I run `silo rm`
      Then the exit code should not be 0
      And the error should indicate "silo-abc12345 is running"
      And the container "silo-abc12345" should still exist

  Rule: --force stops and removes a running container

    Scenario: --force stops running container before removing
      Given the container "silo-abc12345" is running
      When I run `silo rm --force`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
      And podman should run "rm" with "-f" on "silo-abc12345"
      And the exit code should be 0

  Rule: Container not found is a no-op

    Scenario: missing container prints not found and exits 0
      Given no container exists
      When I run `silo rm`
      Then the output should contain "silo-abc12345 not found"
      And the exit code should be 0

  Rule: rm does not remove the image

    Scenario: rm only removes the container, not the image
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then podman should run "rm" on "silo-abc12345"
      But podman should not run "rmi" on "silo-abc12345"
      And the workspace image "silo-abc12345" should still exist

  Rule: Requires workspace to be initialized

    Scenario: rm fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo rm`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing
