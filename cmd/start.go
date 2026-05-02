package cmd

import (
	"github.com/mrkuz/silo/internal"
)

// Start implements `silo start`.
func Start() error {
	_, err := internal.EnsureStarted()
	return err
}
