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
      And the output should contain "Generated .devcontainer.json"
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
      And the output should contain "Generated .devcontainer.json"
      And the exit code should be 0

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
      And the user image "silo-alice" exists
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then shared volume directories should be created before generating .devcontainer.json.

  Rule: User config is merged into generated .devcontainer.json

    Scenario: user devcontainer.in.json merges into generated .devcontainer.json
      Given the user's silo config directory has "devcontainer.in.json" with content '{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should contain the user's "customizations"

    Scenario: arrays are concatenated on merge
      Given the generated .devcontainer.json has "features": ["a", "b"]
      And the user's silo config directory has "devcontainer.in.json" with content '{"features": ["c"]}'
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "features" with all elements "a", "b", "c" in order

    Scenario: scalars from user config override generated values
      Given the user's silo config directory has "devcontainer.in.json" with content '{"name": "my-devcontainer"}'
      And the workspace image "silo-abc12345" exists
      When I run `silo devcontainer`
      Then the .devcontainer.json should have "name" set to "my-devcontainer"

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
