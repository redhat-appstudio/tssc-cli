package subcmd

import "github.com/spf13/cobra"

// Runner controls the "subcommands" workflow from end-to-end, each step of it
// is executed in the predefined sequence: Complete, Validate and Run.
type Runner struct {
	subCmd Interface // SubCommand instance
}

// Cmd exposes the subcommand's cobra command instance.
func (r *Runner) Cmd() *cobra.Command {
	return r.subCmd.Cmd()
}

// NewRunner completes the informed subcommand with the lifecycle methods.
func NewRunner(subCmd Interface) *Runner {
	subCmd.Cmd().PreRunE = func(_ *cobra.Command, args []string) error {
		if err := subCmd.Complete(args); err != nil {
			return err
		}
		return subCmd.Validate()
	}
	subCmd.Cmd().RunE = func(_ *cobra.Command, _ []string) error {
		return subCmd.Run()
	}
	return &Runner{subCmd: subCmd}
}
