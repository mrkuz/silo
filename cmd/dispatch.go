package cmd

var commands = map[string]func([]string) error{
	"init":                 Init,
	"build":                Build,
	"start":                NoArgs(Start),
	"volume setup":         NoArgs(VolumeSetup),
	"connect":              NoArgs(Connect),
	"stop":                 NoArgs(Stop),
	"rm":                   NoArgs(Remove),
	"status":               NoArgs(Status),
	"user init":            UserInit,
	"devcontainer":         DevcontainerGenerate,
	"devcontainer connect": NoArgs(DevcontainerConnect),
	"devcontainer stop":    NoArgs(DevcontainerStop),
	"devcontainer status":  NoArgs(DevcontainerStatus),
}

func Dispatch(args []string) error {
	if len(args) >= 1 {
		if len(args) >= 2 {
			compound := args[0] + " " + args[1]
			if run, ok := commands[compound]; ok {
				return run(args[2:])
			}
		}
		arg := args[0]
		if run, ok := commands[arg]; ok {
			return run(args[1:])
		}
		switch arg {
		case "help", "--help", "-h":
			Help()
			return nil
		default:
			return Run(args)
		}
	}
	return Run(args)
}
