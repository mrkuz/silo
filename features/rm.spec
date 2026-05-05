@rm
Feature: silo rm — Remove the workspace image

  `silo rm` removes the workspace image. If the container exists and is stopped,
  it is removed first. If the container is running, an error is returned and
  neither the container nor the image is touched. Unlike `silo user rm`, this
  removes the per-workspace image (`silo-<id>`), not the shared user image
  (`silo-<user>`).

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Removes the workspace image

    Scenario: rm removes the workspace image when no container exists
      Given no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then podman should run "rmi" on "silo-abc12345"
      And the output should contain "Removing silo-abc12345..."
      And the exit code should be 0

    Scenario: missing image prints not found
      Given no workspace image exists
      When I run `silo rm`
      Then the output should contain "silo-abc12345 not found"
      And the exit code should be 0

  Rule: Running container blocks removal

    Scenario: running container returns error without modifying state
      Given the container "silo-abc12345" is running
      And the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then the output should contain "silo-abc12345 is running"
      And podman should not run "stop" on "silo-abc12345"
      And podman should not run "rm" on "silo-abc12345"
      And podman should not run "rmi" on "silo-abc12345"
      And the exit code should not be 0

  Rule: Stopped container is removed before image removal

    Scenario: stopped container is removed before image removal
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo rm`
      Then podman should not run "stop" on "silo-abc12345"
      And podman should run "rm" on "silo-abc12345"
      And podman should run "rmi" on "silo-abc12345"
      And the exit code should be 0

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

  Rule: Unknown flags show error and help

    Scenario: unknown flag is rejected
      When I run `silo rm --force`
      Then the error should indicate "erroneous command"
      And the error should contain "Usage:"