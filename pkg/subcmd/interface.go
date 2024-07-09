package subcmd

import (
	"github.com/spf13/cobra"
)

// Interface defines the interface for a subcommand, as well the sequence of steps
// each subcommand is obliged to follow.
type Interface interface {
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
