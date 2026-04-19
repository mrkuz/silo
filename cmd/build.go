package cmd

import (
	"fmt"
	"os/user"

	"github.com/mrkuz/silo/internal"
)

// Build implements `silo build`. Builds the workspace image if missing.
func Build() error {
	cfg, _, err := internal.EnsureInit(nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	if err := internal.EnsureImages(cfg); err != nil {
		return err
	}
	return nil
}

// UserBuild implements `silo user build`. Builds the user image if missing.
func UserBuild() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	cfg := internal.Config{
		General: internal.GeneralConfig{
			User:          u.Username,
			ContainerName: "silo-" + u.Username,
		},
	}
	tc, err := internal.NewTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if internal.ImageExists(tc.BaseImage) {
		fmt.Printf("%s already exists\n", tc.BaseImage)
		return nil
	}
	if err := internal.EnsureUserImage(tc); err != nil {
		return err
	}
	return nil
}
