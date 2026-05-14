// silo is a CLI tool for creating per-workspace development containers.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/mrkuz/silo/cmd"
)

func main() {
	if err := cmd.Dispatch(os.Args[1:]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "silo:", unwrapFirst(err))
	os.Exit(1)
}

func unwrapFirst(err error) error {
	for {
		unwrap := errors.Unwrap(err)
		if unwrap == nil {
			return err
		}
		err = unwrap
	}
}
