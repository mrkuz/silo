package internal

import (
	"fmt"
	"os"
)

// PrintInitFileStatus prints the status of an init file.
func PrintInitFileStatus(path string) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("%s already exists\n", path)
		return nil
	} else if os.IsNotExist(err) {
		fmt.Printf("Creating %s...\n", path)
		return nil
	} else {
		return fmt.Errorf("stat %s: %w", path, err)
	}
}
