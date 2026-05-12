@volume_setup
Feature: silo volume setup — Create directories on the persistence volume

  `silo volume setup` creates directories on the persistence volume so they can be mounted
  as subpath volumes inside containers. It runs a temporary container with the workspace
  image — the workspace container does not need to be running. It is also run
  automatically after every `silo start`.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Creates directories on the persistence volume

    Scenario: volume setup creates directories on the persistence volume
      Given the config has shared_paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
      And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
      And the output should contain "volume setup complete"
      And the exit code should be 0

    Scenario: volume setup creates both files and directories
      Given the config has shared_paths ["$HOME/.cache/uv/", "$HOME/.local/share/fish/fish_history"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
      And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
      And the run command should create "/silo/persistence/shared/home/alice/.local/share/fish/fish_history" as a file with mode 644
      And the exit code should be 0

  Rule: No-op when shared paths is empty

    Scenario: empty shared_paths list is a no-op
      Given the config has shared_paths []
      When I run `silo volume setup`
      Then no podman run should be called
      And the output should not contain "volume setup complete"
      And the exit code should be 0

  Rule: Uses workspace image for temporary container

    Scenario: volume setup uses workspace image and does not require workspace container to exist
      Given the config has shared_paths ["$HOME/.cache/uv/"]
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

  Rule: Creates private paths for the silo

    Scenario: volume setup creates private paths under silo-specific directory
      Given the config has private_paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
      And the run command should create "/silo/persistence/abc12345/home/alice/.cache/uv" as a directory with mode 755
      And the output should contain "volume setup complete"
      And the exit code should be 0

    Scenario: volume setup creates both shared and private paths
      Given the config has shared_paths ["$HOME/.local/share/fish/"]
      And the config has private_paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
      And the run command should create "/silo/persistence/shared/home/alice/.local/share/fish" as a directory with mode 755
      And the run command should create "/silo/persistence/abc12345/home/alice/.cache/uv" as a directory with mode 755
      And the output should contain "volume setup complete"
      And the exit code should be 0

    Scenario: empty private_paths list does not create private directories
      Given the config has shared_paths ["$HOME/.cache/uv/"]
      And the config has private_paths []
      And the workspace image "silo-abc12345" exists
      When I run `silo volume setup`
      Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
      And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
      And no private path directory should be created under "/silo/persistence/abc12345/"
      And the output should contain "volume setup complete"
      And the exit code should be 0