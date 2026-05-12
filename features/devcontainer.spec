@devcontainer
Feature: silo devcontainer — Generate a .devcontainer.json for VS Code

  `silo devcontainer` generates a `.devcontainer.json` for VS Code in the current
  directory. It is independent from the main workspace container (silo-<id>) and is
  managed separately by VS Code. The generated devcontainer uses the workspace image
  and the container name is `<workspace-container-name>-dev`.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory
    And the user's silo config directory has all starter files

  Rule: Generates .devcontainer.json

    Scenario: devcontainer generates a .devcontainer.json file
      Given the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then a file ".devcontainer.json" should be created
      And the output should contain "Creating .devcontainer.json"
      And the exit code should be 0

    Scenario: existing .devcontainer.json is not overwritten
      Given the workspace image "silo-abc12345" exists
      And a file ".devcontainer.json" already exists with content '{"name": "custom"}'
      When I run `silo devcontainer`
      Then the file ".devcontainer.json" should still contain '{"name": "custom"}'
      And the output should contain "'.devcontainer.json' already exists"
      And the exit code should be 0

    Scenario: --force overwrites existing .devcontainer.json
      Given the workspace image "silo-abc12345" exists
      And a file ".devcontainer.json" already exists with content '{"name": "custom"}'
      When I run `silo devcontainer --force`
      Then the file ".devcontainer.json" should not contain '{"name": "custom"}'
      And the output should contain "Creating .devcontainer.json"
      And the exit code should be 0

    Scenario: --force does not affect .silo/devcontainer.json handling
      Given the workspace image "silo-abc12345" exists
      And a file ".silo/devcontainer.json" already exists with content '{"custom": true}'
      When I run `silo devcontainer --force`
      Then the output should contain "'.silo/devcontainer.json' already exists"
      And a file ".silo/devcontainer.json" should still contain '{"custom": true}'

    Scenario: unknown flag shows error and help
      When I run `silo devcontainer --unknown`
      Then the stderr should contain "silo: unknown flag \"--unknown\""
      And the stderr should contain "Usage:"
      And the exit code should be 1

    Scenario: devcontainer uses the workspace image
      Given the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should reference image "silo-abc12345"

    Scenario: devcontainer uses a distinct container name
      Given the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should specify container name "silo-abc12345-dev"

    Scenario: devcontainer runs volume setup before generating when shared volume is configured
      Given the config has paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then shared volume directories should be created before generating .devcontainer.json.

    Scenario: devcontainer includes forwardPorts when network ports are configured
      Given a workspace with silo config "abc12345"
      And the config has network ports ["8080:8080", "3000:3000"]
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "forwardPorts" with all elements "8080:8080", "3000:3000" in order

    Scenario: devcontainer omits forwardPorts when no network ports are configured
      Given a workspace with silo config "abc12345"
      And the config has network ports []
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should not have "forwardPorts"

  Rule: Merge order: template wins > .silo > user

    Scenario: template values override user config
      Given the user's silo config directory has "devcontainer.user.json" with content '{"name": "my-devcontainer"}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "name" set to "silo-abc12345-dev"

    Scenario: .silo overrides user config, template wins over both
      Given the user's silo config directory has "devcontainer.user.json" with content '{"name": "user-name", "custom": "user-value"}'
      And the workspace has ".silo/devcontainer.json" with content '{"name": "silo-name", "custom": "silo-value"}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "name" set to "silo-abc12345-dev"
      And the .devcontainer.json should have "custom" set to "silo-value"

    Scenario: arrays from all sources are concatenated in order
      Given the workspace has ".silo/devcontainer.json" with content '{"features": ["silo-feat"]}'
      And the user's silo config directory has "devcontainer.user.json" with content '{"features": ["user-feat"]}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "features" with all elements "user-feat", "silo-feat" in order

    Scenario: user extensions are merged into template customizations
      Given the user's silo config directory has "devcontainer.user.json" with content '{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have customizations.vscode.extensions containing "ms-python.python"

  Rule: Requires workspace to be initialized

    Scenario: devcontainer fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo devcontainer`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing

  Rule: devcontainer is independent from workspace container

    Scenario: devcontainer command does not create the workspace container
      Given the workspace image "silo-abc12345" exists
      And no container exists
      When I run `silo devcontainer`
      Then no workspace container should be created

  Rule: Creates .silo/devcontainer.json boilerplate for project-specific customization

    Scenario: .silo/devcontainer.json is created when not present
      Given the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then a file ".silo/devcontainer.json" should be created
      And the output should contain "Creating .silo/devcontainer.json"

    Scenario: .silo/devcontainer.json is skipped when already present
      Given the workspace image "silo-abc12345" exists
      And a file ".silo/devcontainer.json" already exists
      When I run `silo devcontainer`
      Then the output should contain "'.silo/devcontainer.json' already exists"
      And a file ".silo/devcontainer.json" should not be modified

    Scenario: .silo/devcontainer.json is created even when .devcontainer.json is skipped
      Given the workspace image "silo-abc12345" exists
      And a file ".devcontainer.json" already exists
      When I run `silo devcontainer`
      Then a file ".silo/devcontainer.json" should be created
