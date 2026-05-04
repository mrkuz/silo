package cmd

import (
	"fmt"
	"os/user"

	"github.com/mrkuz/silo/internal"
)

// UserBuild implements `silo user build`. Builds the user image if missing.
func UserBuild(args []string) error {
	force, _, err := ParseForceFlag("user build", args)
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	cfg := internal.Config{
		General: internal.GeneralConfig{
			User: u.Username,
		},
	}
	tc, err := internal.NewTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if err := internal.EnsureUserFiles(); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
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
	return nil
}

// UserRm implements `silo user rm`. Removes the user image.
func UserRm() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	userImage := internal.BaseImageName(u.Username)
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