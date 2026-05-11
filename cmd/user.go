package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// UserInit implements `silo user init`. Prints per-file status
// (for existing and new files) and delegates the actual
// file creation to EnsureUserFiles.
func UserInit(args []string) error {
	files, err := internal.UserStarterFiles()
	if err != nil {
		return fmt.Errorf("list user starter files: %w", err)
	}
	for _, f := range files {
		if err := internal.PrintInitFileStatus(f.Path); err != nil {
			return err
		}
	}
	if err := internal.EnsureUserFiles(); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}
	if _, err := internal.LoadSiloUserTOML(); err != nil {
		return fmt.Errorf("load user configuration: %w", err)
	}
	return nil
}