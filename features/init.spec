@init
Feature: silo init — Initialize workspace

  `silo init` creates workspace configuration and starter files. It is idempotent:
  subsequent runs do not overwrite existing files.

  Background:
    Given a clean workspace with no existing silo files
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: First run creates workspace files

    Scenario: init creates .silo directory with config and home.nix
      When I run `silo init`
      Then a file ".silo/silo.toml" should be created
      And a file ".silo/home.nix" should be created
      And the exit code should be 0

    Scenario: init creates user starter files
      When I run `silo init`
      Then a file "home.user.nix" should be created in the user's silo config directory
      And a file "devcontainer.in.json" should be created in the user's silo config directory
      And a file "silo.user.toml" should be created in the user's silo config directory
      And the exit code should be 0

  Rule: Idempotency — subsequent runs do not modify existing config

    Scenario: existing config is not overwritten
      Given a workspace with silo config "abc12345"
      And the config has id "abc12345"
      When I run `silo init`
      Then the config should still have id "abc12345"
      And the exit code should be 0

    Scenario: existing podman setting is preserved when flag not provided
      Given a workspace with silo config "abc12345"
      And the config has podman=true
      When I run `silo init`
      Then the config should still have podman=true
      And the exit code should be 0

    Scenario: existing podman setting is preserved when flag provided
      Given a workspace with silo config "abc12345"
      And the config has podman=true
      When I run `silo init --no-podman`
      Then the config should still have podman=true
      And the exit code should be 0

  Rule: silo.init creates workspace config from defaults only

    Scenario: init creates workspace config with defaults on first run
      Given a clean workspace with no existing silo files
      When I run `silo init`
      Then the workspace config should have an 8-character random id
      And the workspace config should have podman=false (default)
      And the workspace config should have empty shared volume paths (default)
      And the workspace config should have default create arguments
      And the workspace config should have no user set

    Scenario: feature flags override defaults on first run
      Given a clean workspace with no existing silo files
      When I run `silo init --podman`
      Then the workspace config should have podman=true
      And the file ".silo/home.nix" should contain "silo.podman.enable = true"

    Scenario: silo init does not read silo.user.toml on first run
      Given the user's silo config directory has "silo.user.toml" with content:
        """
        [general]
        user = "alice"

        [shared_volume]
        paths = ["$HOME/.cache/uv/"]
        """
      And a clean workspace with no existing silo files
      When I run `silo init`
      Then the workspace config should have empty shared volume paths (default)
      And the workspace config should have no user set

  Rule: podman feature flag is stored in home.nix

    Scenario: init --podman enables podman in home.nix
      Given a clean workspace with no existing silo files
      When I run `silo init --podman`
      Then the file ".silo/home.nix" should contain "silo.podman.enable = true"

    Scenario: init --no-podman disables podman in home.nix
      Given a clean workspace with no existing silo files
      When I run `silo init --no-podman`
      Then the file ".silo/home.nix" should contain "silo.podman.enable = false"

  Rule: unknown flags show error and help

    Scenario: unknown flag is rejected
      When I run `silo init --unknown`
      Then the stderr should contain "silo: unknown flag"
      And the stderr should contain "Usage:"
      And the exit code should be 1

  Rule: Requires workspace config to be valid

    Scenario: missing id in .silo/silo.toml returns error
      Given a workspace with silo config "abc12345"
      And the config has id ""
      When I run `silo init`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing required field
