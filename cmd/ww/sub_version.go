package main

func versionCmd() command {
	return command{
		name:        "version",
		description: "Print version information",
		fn: func(args []string, glOpts *globalOpts) error {
			printVersion(glOpts.output)
			return nil
		},
	}
}
