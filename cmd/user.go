package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// UserBuild implements `silo user build`. Builds the user image if missing.
func UserBuild(args []string) error {
	force, _, err := ParseForceFlag("user build", args)
	if err != nil {
		return err
	}
	if err := internal.EnsureUserFiles(); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}
	userCfg, err := internal.LoadSiloUserTOML()
	if err != nil {
		return fmt.Errorf("load user config: %w", err)
	}
	cfg := internal.MergedConfig{
		User: userCfg.General.User,
	}
	tc, err := internal.NewTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if !force && internal.ImageExists(tc.BaseImage) {
		fmt.Printf("%s already exists\n", tc.BaseImage)
		return nil
	}
	if err := internal.EnsureUserImage(tc, force); err != nil {
		return err
	}
	return nil
}

// UserRm implements `silo user rm`. Removes the user image.
func UserRm() error {
	if err := internal.EnsureUserFiles(); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}
	userCfg, err := internal.LoadSiloUserTOML()
	if err != nil {
		return fmt.Errorf("load user config: %w", err)
	}
	userImage := internal.BaseImageName(userCfg.General.User)
	if internal.ImageExists(userImage) {
		fmt.Printf("Removing %s...\n", userImage)
		if err := internal.RemoveImage(userImage); err != nil {
			return fmt.Errorf("remove user image: %w", err)
		}
	} else {
		internal.PrintNotFound(userImage)
	}
	return nil
}

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
