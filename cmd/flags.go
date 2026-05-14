package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// NewFlagSet creates a FlagSet configured for silo command parsing.
// It uses ContinueOnError (to allow custom error handling) and suppresses
// the default flag package Usage output.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}
	return fs
}

// ErroneousCommand returns an error for an unrecognized command, flag, or argument.
func ErroneousCommand() error {
	return fmt.Errorf("erroneous command\n\n%s", HelpText)
}

// NoArgs accepts no arguments. It returns an error for any arg.
func NoArgs(f func() error) func([]string) error {
	return func(args []string) error {
		if len(args) == 0 {
			return f()
		}
		return ErroneousCommand()
	}
}

// parseWithInterceptor runs fs.Parse() while suppressing the default
// "flag provided but not defined" message from Go's flag package.
func parseWithInterceptor(fs *flag.FlagSet, args []string) error {
	if len(args) == 0 {
		return nil
	}

	// Use a pipe to suppress stderr where flag package writes its error
	r, w, err := os.Pipe()
	if err != nil {
		return fs.Parse(args)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() {
		w.Close()
		os.Stderr = oldStderr
		io.Copy(io.Discard, r)
		r.Close()
	}()

	err = fs.Parse(args)

	if err == nil {
		return nil
	}

	return ErroneousCommand()
}

func DetectConflictingFlags(fs *flag.FlagSet, flag1, flag2 string) error {
	f1 := fs.Lookup(flag1)
	f2 := fs.Lookup(flag2)
	if f1 != nil && f2 != nil && f1.Value.String() == "true" && f2.Value.String() == "true" {
		return fmt.Errorf("conflicting flags: --%s and --%s", flag1, flag2)
	}
	return nil
}
