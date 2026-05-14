@build
Feature: silo build — Build workspace images

  `silo build` ensures the workspace image exists,
  building it if missing. It runs `silo init` implicitly first.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory
    And the user's silo config directory has all starter files

  Rule: Builds workspace image when missing

    Scenario: build creates workspace image
      Given no workspace image exists
      When I run `silo build`
      Then the workspace image "silo-abc12345" should be built
      And the exit code should be 0

    Scenario: build prints build message
      Given no workspace image exists
      When I run `silo build`
      Then the output should contain "Building workspace image silo-abc12345..."

  Rule: Idempotency — existing images are skipped

    Scenario: workspace image exists is a no-op
      Given the workspace image "silo-abc12345" exists
      When I run `silo build`
      Then the output should contain "silo-abc12345 already exists"
      And no build should occur
      And the exit code should be 0

  Rule: Init on demand — build initializes workspace if not initialized

    Scenario: build creates workspace config if missing
      Given a clean workspace with no existing silo files
      And the user's silo config directory has all starter files
      And no workspace image exists
      When I run `silo build`
      Then a file ".silo/silo.toml" should be created
      And the workspace image "silo-abc12345" should be built
      And the exit code should be 0

  Rule: home.nix is baked into the workspace image

    Scenario: workspace home.nix content is included in the built image
      Given the workspace has "home.nix" with content:
        """
        home.packages = with pkgs; [ nodejs python3 ];
        """
      And no workspace image exists
      When I run `silo build`
      Then the workspace image build should include a file "home.nix" containing "nodejs python3"

  Rule: --rebuild forces workspace image rebuild

    Scenario: build --rebuild rebuilds even when image exists
      Given the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" does not exist
      When I run `silo build --rebuild`
      Then the workspace image "silo-abc12345" should be built

    Scenario: build --rebuild aborts if container is running
      Given the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" is running
      When I run `silo build --rebuild`
      Then the exit code should not be 0
      And the error should contain "running"

    Scenario: build --rebuild aborts if container exists (stopped)
      Given the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" exists but is stopped
      When I run `silo build --rebuild`
      Then the exit code should not be 0
      And the error should contain "exists"

  Rule: --no-cache disables build cache

    Scenario: build --no-cache builds without cache
      Given no workspace image exists
      When I run `silo build --no-cache`
      Then the workspace image "silo-abc12345" should be built
      And podman build should be called with "--no-cache" for the workspace image

    Scenario: build --rebuild --no-cache rebuilds without cache
      Given the workspace image "silo-abc12345" exists
      And the container "silo-abc12345" does not exist
      When I run `silo build --rebuild --no-cache`
      Then the workspace image "silo-abc12345" should be built
      And podman build should be called with "--no-cache" for the workspace image

  Rule: unknown flags show error and help

    Scenario: unknown flag is rejected
      When I run `silo build --unknown`
      Then the stderr should contain "silo: unknown flag"
      And the stderr should contain "Usage:"
      And the exit code should be 1