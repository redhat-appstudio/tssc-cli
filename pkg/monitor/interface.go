package monitor

import (
	"context"
	"time"

	"k8s.io/cli-runtime/pkg/resource"
)

// Interface is the interface that defines the methods of a Monitor.
type Interface interface {
	// Collect collects relevant resources for later inspection.
	Collect(context.Context, *resource.Info) error

	// Watch waits for all monitoring functions to complete, or until the timeout
	// is reached.
	Watch(time.Duration) error
}
