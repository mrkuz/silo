package cmd

// ParseForceFlag extracts -f/--force and returns remaining args.
// It tolerates unknown flags so callers can pass their own FlagSet on the remainder.
func ParseForceFlag(args []string) (force bool, remaining []string) {
	for _, arg := range args {
		if arg == "-f" || arg == "--force" {
			force = true
			continue
		}
		remaining = append(remaining, arg)
	}
	return force, remaining
}
