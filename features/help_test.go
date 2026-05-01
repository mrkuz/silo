package features_test

import (
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
)

// Feature: silo help — Show command reference
// `silo help` prints the full command reference. It can also be triggered with
// `--help` or `-h` flags on any command.
func TestFeatureHelp(t *testing.T) {
	t.Run("Scenario: help prints the command reference", func(t *testing.T) {
		// When I run `silo help`
		output := cmd.PrintHelp()

		// Then the output should contain "silo - developer sandbox container"
		if !strings.Contains(output, "silo - developer sandbox container") {
			t.Errorf("expected output to contain 'silo - developer sandbox container', got: %s", output)
		}
		// And the output should contain "Usage:"
		if !strings.Contains(output, "Usage:") {
			t.Errorf("expected output to contain 'Usage:', got: %s", output)
		}
		// And the output should contain "silo init"
		if !strings.Contains(output, "silo init") {
			t.Errorf("expected output to contain 'silo init', got: %s", output)
		}
		// And the output should contain "silo build"
		if !strings.Contains(output, "silo build") {
			t.Errorf("expected output to contain 'silo build', got: %s", output)
		}
		// And the output should contain "silo connect"
		if !strings.Contains(output, "silo connect") {
			t.Errorf("expected output to contain 'silo connect', got: %s", output)
		}
		// And the output should contain "silo help"
		if !strings.Contains(output, "silo help") {
			t.Errorf("expected output to contain 'silo help', got: %s", output)
		}
	})

	t.Run("Scenario: --help flag on silo prints the command reference", func(t *testing.T) {
		// When I run `silo --help`
		output := cmd.PrintHelp()

		// Then the output should contain "silo - developer sandbox container"
		if !strings.Contains(output, "silo - developer sandbox container") {
			t.Errorf("expected output to contain 'silo - developer sandbox container', got: %s", output)
		}
	})

	t.Run("Scenario: -h flag on silo prints the command reference", func(t *testing.T) {
		// When I run `silo -h`
		output := cmd.PrintHelp()

		// Then the output should contain "silo - developer sandbox container"
		if !strings.Contains(output, "silo - developer sandbox container") {
			t.Errorf("expected output to contain 'silo - developer sandbox container', got: %s", output)
		}
	})
}
