@user_rmi
Feature: silo user rmi — Remove the shared user image

  `silo user rmi` removes the shared user image (`silo-<username>`) for the
  current user. It has no effect if the image does not exist.

  Background:
    Given the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Removes the user image when present

    Scenario: user rmi removes the user image
      Given the user image "silo-alice" exists
      When I run `silo user rmi`
      Then the user image "silo-alice" should be removed
      And the output should contain "Removing silo-alice..."
      And the exit code should be 0

    Scenario: user rmi prints removal message
      Given the user image "silo-alice" exists
      When I run `silo user rmi`
      Then the output should contain "Removing silo-alice..."

  Rule: Idempotency — image not found is not an error

    Scenario: missing user image is a no-op
      Given no user image exists for the current user
      When I run `silo user rmi`
      Then the output should contain "silo-alice not found"
      And the exit code should be 0

    Scenario: missing user image does not call podman rmi
      Given no user image exists for the current user
      When I run `silo user rmi`
      Then no podman rmi call should be made
      And the exit code should be 0
