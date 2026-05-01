@build
Feature: silo build — Build workspace images

  `silo build` ensures both the user image and the workspace image exist,
  building either or both if missing. It runs `silo init` implicitly first.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory
    And the user's silo config directory has all starter files

  Rule: Builds both images when both are missing

    Scenario: build creates user image first, then workspace image
      Given no user image exists
      And no workspace image exists
      When I run `silo build`
      Then the user image "silo-alice" should be built
      And the workspace image "silo-abc12345" should be built
      And the exit code should be 0

    Scenario: build prints build messages in order
      Given no user image exists
      And no workspace image exists
      When I run `silo build`
      Then the output should contain "Building user image silo-alice..."
      And the output should contain "Building workspace image silo-abc12345..."

  Rule: Idempotency — existing images are skipped

    Scenario: both images exist is a no-op
      Given the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo build`
      Then the output should contain "silo-alice already exists"
      And the output should contain "silo-abc12345 already exists"
      And no build should occur
      And the exit code should be 0

    Scenario: user image exists, workspace missing
      Given the user image "silo-alice" exists
      And no workspace image exists
      When I run `silo build`
      Then the user image should not be rebuilt
      And the workspace image "silo-abc12345" should be built
      And the output should contain "Building workspace image silo-abc12345..."

  Rule: Init on demand — build initializes workspace if not initialized

    Scenario: build creates workspace config if missing
      Given a clean workspace with no existing silo files
      And the user's XDG_CONFIG_HOME points to a fresh directory
      And the user's silo config directory has all starter files
      And no user image exists
      And no workspace image exists
      When I run `silo build`
      Then a file ".silo/silo.toml" should be created
      And images should be built
      And the exit code should be 0

  Rule: home.nix is baked into the workspace image

    Scenario: workspace home.nix content is included in the built image
      Given a workspace with silo config "abc12345"
      And the workspace has "home.nix" with content:
        """
        home.packages = with pkgs; [ nodejs python3 ];
        """
      And no user image exists
      And no workspace image exists
      When I run `silo build`
      Then the workspace image build should include a file "home-workspace.nix" containing "nodejs python3"

  Rule: --force forces workspace image rebuild

    Scenario: build --force rebuilds even when image exists
      Given the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" does not exist
      When I run `silo build --force`
      Then the workspace image "silo-abc12345" should be built

    Scenario: build --force aborts if container is running
      Given the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" is running
      When I run `silo build --force`
      Then the exit code should not be 0
      And the error should contain "running"

    Scenario: build --force aborts if container exists (stopped)
      Given the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" exists but is stopped
      When I run `silo build --force`
      Then the exit code should not be 0
      And the error should contain "exists"
