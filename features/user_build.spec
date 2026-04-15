@user_build
Feature: silo user build — Build the shared user image

  `silo user build` builds the shared user image (`silo-<username>`) if it does not
  already exist. The user image is shared across all workspaces and includes the
  user's `home-user.nix`. It is a prerequisite for workspace image builds.

  Background:
    Given the user's XDG_CONFIG_HOME points to a fresh directory
    And the user's silo config directory has all starter files

  Rule: Builds the user image if missing

    Scenario: missing user image triggers build
      Given no user image exists for the current user
      When I run `silo user build`
      Then the user image "silo-alice" should be built
      And the exit code should be 0

    Scenario: build prints a message while building
      Given no user image exists for the current user
      When I run `silo user build`
      Then the output should contain "Building user image silo-alice..."
      And the exit code should be 0

  Rule: Idempotency — existing image is not rebuilt

    Scenario: existing user image is skipped
      Given the user image "silo-alice" already exists
      When I run `silo user build`
      Then the output should contain "silo-alice already exists"
      And no build should occur
      And the exit code should be 0

  Rule: Requires home-user.nix to be present

    Scenario: build fails if home-user.nix is missing
      Given the user image "silo-alice" does not exist
      But the user's silo config directory is missing "home-user.nix"
      When I run `silo user build`
      Then the exit code should not be 0
      And the error should indicate "home-user.nix" could not be read

  Rule: home-user.nix is baked into the user image

    Scenario: user's home-user.nix content is included in the built image
      Given the user's silo config directory has "home-user.nix" with content:
        """
        home.packages = with pkgs; [ vim git ];
        """
      And no user image exists
      When I run `silo user build`
      Then the podman build context should include a file "home-user.nix" containing "vim git"
