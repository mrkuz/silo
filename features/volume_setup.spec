@volume_setup
Feature: silo volume setup — Create directories on the shared volume

  `silo volume setup` creates directories on the shared volume so they can be mounted
  as subpath volumes inside containers. It runs a temporary container with the workspace
  image — the workspace container does not need to be running. It is also run
  automatically after every `silo start`.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Creates directories on the shared volume

    Scenario: volume setup creates directories on the shared volume
      Given the config has paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo-shared:/silo/shared:z"
      And the run command should create "/silo/shared/home/alice/.cache/uv" as a directory with mode 755
      And the output should contain "volume setup complete"
      And the exit code should be 0

    Scenario: volume setup creates both files and directories
      Given the config has paths ["$HOME/.cache/uv/", "$HOME/.local/share/fish/fish_history"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo-shared:/silo/shared:z"
      And the run command should create "/silo/shared/home/alice/.cache/uv" as a directory with mode 755
      And the run command should create "/silo/shared/home/alice/.local/share/fish/fish_history" as a file with mode 644
      And the exit code should be 0

  Rule: No-op when shared volume paths is empty

    Scenario: empty paths list is a no-op
      Given the config has paths []
      When I run `silo volume setup`
      Then no podman run should be called
      And the output should not contain "volume setup complete"
      And the exit code should be 0

  Rule: Uses workspace image for temporary container

    Scenario: volume setup uses workspace image and does not require workspace container to exist
      Given the config has paths ["$HOME/.cache/uv/"]
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then the output should contain "volume setup complete"
      And the exit code should be 0

  Rule: Requires workspace to be initialized

    Scenario: volume setup fails when workspace is not initialized
      Given a clean workspace with no existing silo files
      When I run `silo volume setup`
      Then the exit code should not be 0
      And the error should indicate ".silo/silo.toml" is missing