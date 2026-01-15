package api

import (
	"github.com/spf13/cobra"
)

// SubCommand defines the interface for a subcommand, as well the sequence of
// steps every cobra.Command is expected to follow.
type SubCommand interface {
	Cmd() *cobra.Command

	// Complete loads the external dependencies for the subcommand, such as
	// configuration files or checking the Kubernetes API client connectivity.
	Complete(_ []string) error

	// Validate checks the subcommand configuration, asserts the required fields
	// are valid before running the primary action.
	Validate() error

	// Run executes the subcommand "business logic".
	Run() error
}

// Runner controls the "subcommands" workflow from end-to-end, each step of it
// is executed in the predefined sequence: Complete, Validate and Run.
type Runner struct {
	subCmd SubCommand // SubCommand instance
}

// Cmd exposes the subcommand's cobra command instance.
func (r *Runner) Cmd() *cobra.Command {
	return r.subCmd.Cmd()
}

// NewRunner completes the informed subcommand with the lifecycle methods.
func NewRunner(subCmd SubCommand) *Runner {
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
