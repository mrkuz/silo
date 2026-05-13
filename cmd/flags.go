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

// ParseRebuildAndNoCache extracts --rebuild and --no-cache flags and returns remaining args.
func ParseRebuildAndNoCache(cmdName string, args []string) (rebuild bool, noCache bool, remaining []string, err error) {
	fs := NewFlagSet(cmdName)
	rebuildFlag := fs.Bool("rebuild", false, "")
	noCacheFlag := fs.Bool("no-cache", false, "")
	if err := parseWithInterceptor(fs, args); err != nil {
		return false, false, nil, err
	}
	remaining = fs.Args()
	if len(remaining) > 0 {
		return false, false, nil, ErroneousCommand()
	}
	return *rebuildFlag, *noCacheFlag, remaining, nil
}

// ParseUpdateFlag extracts --update and returns remaining args.
func ParseUpdateFlag(cmdName string, args []string) (update bool, remaining []string, err error) {
	fs := NewFlagSet(cmdName)
	updateFlag := fs.Bool("update", false, "")
	if err := parseWithInterceptor(fs, args); err != nil {
		return false, nil, err
	}
	remaining = fs.Args()
	if len(remaining) > 0 {
		return false, nil, ErroneousCommand()
	}
	return *updateFlag, remaining, nil
}
