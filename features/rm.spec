@rm
Feature: silo rm — Remove the workspace image

  `silo rm` removes the workspace image. With `--force`, it also stops and removes
  the container first if it is running. Unlike `silo user rm`, this removes the
  per-workspace image (`silo-<id>`), not the shared user image (`silo-<user>`).

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Removes the workspace image

    Scenario: rm removes the workspace image
      Given the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then podman should run "rmi" on "silo-abc12345"
      And the output should contain "Removing silo-abc12345..."
      And the exit code should be 0

    Scenario: missing image prints not found
      Given no workspace image exists
      When I run `silo rm`
      Then the output should contain "silo-abc12345 not found"
      And the exit code should be 0

  Rule: --force stops and removes container before removing image

    Scenario: --force stops running container before removing image
      Given the container "silo-abc12345" is running
      And the workspace image "silo-abc12345" exists
      When I run `silo rm --force`
      Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
      And podman should run "rm" with "-f" on "silo-abc12345"
      And podman should run "rmi" on "silo-abc12345"
      And the exit code should be 0

    Scenario: --force with stopped container removes image directly
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo rm --force`
      Then podman should run "rmi" on "silo-abc12345"
      But podman should not run "stop" on "silo-abc12345"
      And the exit code should be 0

    Scenario: --force with absent container removes image without trying to stop
      Given no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo rm --force`
      Then podman should run "rmi" on "silo-abc12345"
      But podman should not run "stop" on "silo-abc12345"
      And the exit code should be 0

  Rule: Without --force, running container blocks image removal

    Scenario: running container without --force returns an error
      Given the container "silo-abc12345" is running
      And the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then the exit code should not be 0
      And the error should indicate "silo-abc12345 is running"
      And the output should not contain "Removing"
      And the workspace image "silo-abc12345" should still exist

  Rule: Requires workspace to be initialized

    Scenario: rm fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo rm`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing

  Rule: rm does not remove the user image

    Scenario: rm only removes the workspace image, not the user image
      Given the workspace image "silo-abc12345" exists
      And the user image "silo-alice" exists
      When I run `silo rm`
      Then podman should run "rmi" on "silo-abc12345"
      But podman should not run "rmi" on "silo-alice"
