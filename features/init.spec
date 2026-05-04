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
      And a file "silo.in.toml" should be created in the user's silo config directory
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

  Rule: silo.in.toml seeds new workspace config on first run

    Scenario: silo.in.toml values seed the workspace config
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        [features]
        podman = true

        [shared_volume]
        name = "my-shared"
        paths = ["$HOME/.cache/uv/"]

        [podman]
        create_args = ["--memory=2g"]
        """
      When I run `silo init`
      Then the workspace config should have podman=true
      And the workspace config should have shared_volume name "my-shared"
      And the workspace config should have create arguments ["--memory=2g"]

    Scenario: silo.in.toml [general] section is ignored
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        [general]
        id = "ignored-id"
        user = "ignored-user"
        """
      When I run `silo init`
      Then the workspace config should have an 8-character random id
      And the workspace config should use the current username

    Scenario: silo.in.toml empty or absent uses built-in defaults
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        """
      When I run `silo init`
      Then the workspace config should have podman=false
      And the workspace config should have shared_volume name "silo-shared"

    Scenario: silo.in.toml is created if it does not exist
      Given the user's silo config directory exists but "silo.in.toml" is absent
      When I run `silo init`
      Then a file "silo.in.toml" should be created in the user's silo config directory
      And the file "silo.in.toml" in the user's silo config directory should be empty

    Scenario: silo.in.toml create arguments are prepended to default arguments
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        [podman]
        create_args = ["--memory=2g"]
        """
      When I run `silo init`
      Then the workspace config should have 5 create arguments
      And the first create argument should be "--memory=2g"
      And the second create argument should be "--cap-drop=ALL"

  Rule: Feature flags set initial config on first run

    Scenario: --podman sets podman=true on first run
      Given a clean workspace with no existing silo files
      When I run `silo init --podman`
      Then the workspace config should have podman=true
      And the file ".silo/home.nix" should contain "silo.podman.enable = true"
      And the exit code should be 0

    Scenario: --no-podman sets podman=false on first run
      Given a clean workspace with no existing silo files
      When I run `silo init --no-podman`
      Then the workspace config should have podman=false
      And the file ".silo/home.nix" should not contain "silo.podman.enable = true"
      And the exit code should be 0

    Scenario: --podman flag overrides seeded config from silo.in.toml on first run
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        [features]
        podman = true
        """
      And a clean workspace with no existing silo files
      When I run `silo init --no-podman`
      Then the workspace config should have podman=false
      And the exit code should be 0

  Rule: Feature flags are ignored on subsequent runs without --force

    Scenario: --podman does not modify config on subsequent run
      Given a workspace with silo config "abc12345"
      And the config has podman=false
      When I run `silo init --podman`
      Then the config should have podman=false
      And the exit code should be 0

    Scenario: --no-podman does not modify config on subsequent run
      Given a workspace with silo config "abc12345"
      And the config has podman=true
      When I run `silo init --no-podman`
      Then the config should have podman=true
      And the exit code should be 0

  Rule: Explicit flags only affect config with --force

    Scenario: --podman enables podman feature only with --force
      Given a workspace with silo config "abc12345"
      And the config has podman=false
      When I run `silo init --force --podman`
      Then the config should have podman=true
      And the exit code should be 0

    Scenario: --no-podman disables podman feature only with --force
      Given a workspace with silo config "abc12345"
      And the config has podman=true
      When I run `silo init --force --no-podman`
      Then the config should have podman=false
      And the exit code should be 0

    Scenario: --podman flag overrides seeded config from silo.in.toml only with --force
      Given the user's silo config directory has "silo.in.toml" with content:
        """
        [features]
        podman = true
        """
      And a clean workspace with no existing silo files
      When I run `silo init --force --no-podman`
      Then the workspace config should have podman=false
      And the exit code should be 0

  Rule: Podman flag affects workspace home.nix only with --force

    Scenario: --podman adds podman module to home.nix only with --force
      Given a clean workspace with no existing silo files
      When I run `silo init --force --podman`
      Then the file ".silo/home.nix" should contain "silo.podman.enable = true"
      And the exit code should be 0

    Scenario: --no-podman does not add podman module to home.nix only with --force
      Given a clean workspace with no existing silo files
      When I run `silo init --force --no-podman`
      Then the file ".silo/home.nix" should not contain "silo.podman.enable = true"
      And the exit code should be 0

  Rule: Conflicting flags use last value

    Scenario: both --podman and --no-podman uses last flag
      When I run `silo init --podman --no-podman`
      Then the config should have podman=false
      And the exit code should be 0

  Rule: --force overwrites existing workspace files

    Scenario: init --force rewrites existing silo.toml and home.nix
      Given a workspace with silo config "abc12345"
      And the user image "silo-alice" exists
      When I run `silo init --force`
      Then the workspace file ".silo/silo.toml" should be overwritten
      And the workspace file ".silo/home.nix" should be overwritten
      But the config "[general]" section should be preserved
      And the user file "$XDG_CONFIG_HOME/silo/home.user.nix" should not be overwritten

    Scenario: init --force seeds non-[general] from silo.in.toml
      Given a workspace with silo config "abc12345"
      And the config has podman=false
      And the user's silo config directory has "silo.in.toml" with content:
        """
        [features]
        podman = true

        [shared_volume]
        name = "my-shared"
        paths = ["$HOME/.cache/uv/"]
        """
      When I run `silo init --force`
      Then the config should have podman=true
      And the config should have shared_volume name "my-shared"
      And the config should still have id "abc12345"

    Scenario: init --force adds default create arguments
      Given a workspace with silo config "abc12345"
      And the config has podman=false
      And the user's silo config directory has "silo.in.toml" with content:
        """
        [podman]
        create_args = ["--memory=2g"]
        """
      When I run `silo init --force`
      Then the workspace config should have 5 create arguments
      And the first create argument should be "--memory=2g"
      And the second create argument should be "--cap-drop=ALL"

  Rule: Display of file status during init

    Scenario: init shows creating message for new files
      Given a clean workspace with no existing silo files
      When I run `silo init`
      Then the output should contain "Creating .silo/silo.toml"
      And the output should contain "Creating .silo/home.nix"

    Scenario: init shows already exists message for existing files
      Given a workspace with silo config "abc12345"
      When I run `silo init`
      Then the output should contain "'/path/to/workspace/.silo/silo.toml' already exists"
