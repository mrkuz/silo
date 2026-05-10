@user_init
Feature: silo user init — Create user starter files

  `silo user init` creates user-level starter files under `$XDG_CONFIG_HOME/silo/` if
  they do not already exist. It is idempotent: subsequent runs do not overwrite
  existing files.

  Background:
    Given the user's XDG_CONFIG_HOME points to a fresh directory

  Rule: First run creates all user starter files

    Scenario: user init creates all three user files
      When I run `silo user init`
      Then a file "home.user.nix" should be created in the user's silo config directory
      And a file "devcontainer.in.json" should be created in the user's silo config directory
      And a file "silo.user.toml" should be created in the user's silo config directory
      And the exit code should be 0

    Scenario: home.user.nix contains default shell command module
      When I run `silo user init`
      Then the file "home.user.nix" in the user's silo config directory should contain "silo.shellCommand"
      And the file "home.user.nix" in the user's silo config directory should contain "/bin/bash --login"
      And the exit code should be 0

    Scenario: silo.user.toml contains [general] with current username
      When I run `silo user init`
      Then the file "silo.user.toml" in the user's silo config directory should contain `[general]`
      And the file "silo.user.toml" in the user's silo config directory should contain the current username
      And the exit code should be 0

  Rule: Idempotency — existing files are not overwritten

    Scenario: all existing user files are preserved
      Given the user's silo config directory already has "home.user.nix" with content "# custom content"
      And the user's silo config directory already has "devcontainer.in.json" with content "{ \"custom\": true }"
      And the user's silo config directory already has "silo.user.toml" with content "[features]"
      When I run `silo user init`
      Then the file "home.user.nix" in the user's silo config directory should contain "# custom content"
      And the file "devcontainer.in.json" in the user's silo config directory should contain "{ \"custom\": true }"
      And the file "silo.user.toml" in the user's silo config directory should contain "[features]"
      And the exit code should be 0

  Rule: Requires valid user config

    Scenario: missing user in silo.user.toml returns error
      Given the user's silo config directory has "silo.user.toml" with content "[features]"
      When I run `silo user init`
      Then the exit code should not be 0
      And the error should indicate "[general].user is required"

    Scenario: user init shows already exists message for existing files
      Given the user's silo config directory already has all starter files
      When I run `silo user init`
      Then the output should contain "'<XDG_CONFIG_HOME>/silo/home.user.nix' already exists"
      And the output should contain "'<XDG_CONFIG_HOME>/silo/devcontainer.in.json' already exists"
      And the output should contain "'<XDG_CONFIG_HOME>/silo/silo.user.toml' already exists"
