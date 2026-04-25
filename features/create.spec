@create
Feature: silo create — Create a workspace container

  `silo create` builds images if needed and creates a Podman container from the
  workspace image. The container is left stopped. Subsequent runs skip creation
  if the container already exists.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory
    And the user image "silo-alice" exists
    And the workspace image "silo-abc12345" exists

  Rule: Creates the container from the workspace image

    Scenario: create makes the container
      When I run `silo create`
      Then a Podman container "silo-abc12345" should be created
      And the exit code should be 0

    Scenario: create does not start the container
      When I run `silo create`
      Then the container "silo-abc12345" should exist but not be running

    Scenario: create prints a creation message
      When I run `silo create`
      Then the output should contain "Creating silo-abc12345..."

  Rule: Idempotency — container is not recreated if it already exists

    Scenario: existing container is not overwritten
      Given the container "silo-abc12345" already exists
      When I run `silo create`
      Then the output should contain "silo-abc12345 already exists"
      And no new container should be created
      And the exit code should be 0

  Rule: --dry-run prints the podman create command without running it

    Scenario: dry-run shows full podman create command
      When I run `silo create --dry-run`
      Then the output should contain "podman create"
      And the output should contain "--name" and "silo-abc12345"
      And the output should contain "--hostname" and "silo-abc12345"

    Scenario: dry-run does not create a container
      Given no container exists
      When I run `silo create --dry-run`
      Then the container "silo-abc12345" should not exist
      And the exit code should be 0

    Scenario: dry-run shows workspace mount
      When I run `silo create --dry-run`
      Then the output should contain "--volume"
      And the output should contain "--workdir"

    Scenario: dry-run works even if container already exists
      Given the container "silo-abc12345" already exists
      When I run `silo create --dry-run`
      Then the output should contain "podman create"
      And the output should contain "--name" and "silo-abc12345"
      And the container "silo-abc12345" should still exist unchanged
      And the exit code should be 0

  Rule: Builds images if missing before creating container

    Scenario: missing user image triggers build
      Given no user image exists
      When I run `silo create`
      Then the user image "silo-alice" should be built
      And a Podman container "silo-abc12345" should be created

    Scenario: missing workspace image triggers build
      Given no workspace image exists
      When I run `silo create`
      Then the workspace image "silo-abc12345" should be built
      And a Podman container "silo-abc12345" should be created

  Rule: Uses create arguments from config

    Scenario: podman-enabled config passes security-opt and device flags
      Given a workspace with silo config "abc12345"
      And the config has podman=true
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should include "--security-opt"
      And the podman create command should include "label=disable"
      And the podman create command should include "--device"
      And the podman create command should include "/dev/fuse"

    Scenario: podman-disabled config passes cap-drop, cap-add, and security-opt flags
      Given a workspace with silo config "abc12345"
      And the config has podman=false
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should include "--cap-drop=ALL"
      And the podman create command should include "--cap-add=NET_BIND_SERVICE"
      And the podman create command should include "--security-opt" and "no-new-privileges"

    Scenario: custom create arguments from config are passed to podman
      Given a workspace with silo config "abc12345"
      And the config has create arguments ["--memory=2g", "--cpus=4"]
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should include "--memory=2g"
      And the podman create command should include "--cpus=4"

  Rule: Shared volume mounts when feature is enabled

    Scenario: shared volume is mounted when enabled
      Given a workspace with silo config "abc12345"
      And the config has shared_volume=true
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should include "--mount" with "type=volume" and "source=silo-shared" and "target=/silo/shared" and ",Z"

    Scenario: shared volume paths are mounted as subpath volumes
      Given a workspace with silo config "abc12345"
      And the config has shared_volume=true with paths ["$HOME/.cache/uv/"]
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should include "--mount" with "type=volume,source=silo-shared,target=/silo/shared,subpath=home/alice/.cache/uv,Z"

    Scenario: shared volume is not mounted when disabled
      Given a workspace with silo config "abc12345"
      And the config has shared_volume=false
      And the workspace image "silo-abc12345" exists
      When I run `silo create`
      Then the podman create command should not include "/silo/shared"
