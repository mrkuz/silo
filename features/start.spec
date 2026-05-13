@start
Feature: silo start — Start the workspace container

  `silo start` ensures the container is running. It builds images and creates
  the container if needed, then starts it. If the container is already running,
  it is a no-op. Unlike the default silo invocation, it does not attach to the
  container — it returns after starting.

  Background:
    Given a workspace with silo config "abc12345"
    And the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: Starts the container

    Scenario: start runs podman start
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then podman should run "start" on "silo-abc12345"
      And the exit code should be 0

    Scenario: start prints a message when starting
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the output should contain "Starting silo-abc12345..."

  Rule: Idempotency — already running container is a no-op

    Scenario: running container is not restarted
      Given the container "silo-abc12345" is running
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then no podman start should be called
      And the exit code should be 0

  Rule: Creates container if missing (builds images if needed)

    Scenario: missing container triggers full build-and-create chain
      Given no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And the container "silo-abc12345" should be running
      And the exit code should be 0

    Scenario: missing image triggers build before container creation
      Given no container exists
      And no workspace image exists
      When I run `silo start`
      Then the workspace image "silo-abc12345" should be built
      And the container "silo-abc12345" should be created
      And the container "silo-abc12345" should be running
      And the exit code should be 0

    Scenario: workspace ports are passed to podman create
      Given a workspace with silo config "abc12345"
      And the config has network ports ["8080:8080"]
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "-p 8080:8080"

    Scenario: limits cpus is passed to podman create
      Given a workspace with silo config "abc12345"
      And the config has limits with cpus "2"
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "--cpus=2"

    Scenario: limits memory is passed to podman create
      Given a workspace with silo config "abc12345"
      And the config has limits with memory "4096"
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "--memory=4096"

    Scenario: limits processes is passed to podman create
      Given a workspace with silo config "abc12345"
      And the config has limits with processes "1024"
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "--pids-limit=1024"

    Scenario: zero or negative limits are passed with unlimited values
      Given a workspace with silo config "abc12345"
      And the config has limits with cpus "0" and memory "-1" and processes "0"
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "--cpus=0"
      And podman create should include "--memory=0"
      And podman create should include "--pids-limit=-1"

    Scenario: all limits are passed together
      Given a workspace with silo config "abc12345"
      And the config has limits with cpus "4" and memory "8192" and processes "2048"
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "--cpus=4"
      And podman create should include "--memory=8192"
      And podman create should include "--pids-limit=2048"

    Scenario: multiple ports are all passed to podman create
      Given a workspace with silo config "abc12345"
      And the config has network ports ["8080:8080", "3000:3000"]
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should include "-p 8080:8080"
      And podman create should include "-p 3000:3000"

    Scenario: empty ports array adds no -p arguments
      Given a workspace with silo config "abc12345"
      And the config has network ports []
      And no container exists
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then the container "silo-abc12345" should be created
      And podman create should not include any "-p" arguments

  Rule: Runs volume setup before starting

    Scenario: shared volume directories are created before container starts
      Given a workspace with silo config "abc12345"
      And the config has paths ["$HOME/.cache/uv/"]
      And the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then shared volume directories should be created before the container starts

  Rule: Does not connect to the container

    Scenario: start does not attach to the container
      Given the container "silo-abc12345" exists but is stopped
      And the workspace image "silo-abc12345" exists
      When I run `silo start`
      Then no podman exec should be called
      And the exit code should be 0