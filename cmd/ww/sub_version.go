package main

import "github.com/spf13/pflag"

func versionCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" version", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output JSON")

	return command{
		name:        "version",
		description: "Print version information",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag

			if glOpts.json {
				return outputJSON(glOpts.output, currentVersionInfo())
			}
			printVersion(glOpts.output)
			return nil
		},
	}
}
