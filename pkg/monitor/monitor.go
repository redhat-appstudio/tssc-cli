package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"k8s.io/cli-runtime/pkg/resource"
)

// monitorQueueFn is a function type for monitoring a specific resource.
type monitorQueueFn func() error

// Monitor is the monitoring actor which collects interesting resources from a
// Helm Chart release payload, and monitors them until they are ready. The
// monitoring is executed with a queue of functions, which are executed in order
// until the queue is empty or timeout is reached.
type Monitor struct {
	logger *slog.Logger  // application logger
	kube   k8s.Interface // kubernetes client

	queue []monitorQueueFn // monitor function queue
}

var _ Interface = &Monitor{}

// Collect inspects the resource and adds a monitoring function to the queue.
func (m *Monitor) Collect(ctx context.Context, r *resource.Info) error {
	if r.Object == nil {
		return fmt.Errorf("resource object is nil")
	}

	gvk := r.Object.GetObjectKind().GroupVersionKind()
	gv := gvk.GroupVersion().String()

	logger := m.logger.With(
		"gv", gv,
		"kind", gvk.Kind,
		"name", r.Name,
		"namespace", r.Namespace,
	)
	logger.Debug("Inspecting resource for monitoring...")

	switch fmt.Sprintf("%s/%s", gv, gvk.Kind) {
	case "project.openshift.io/v1/ProjectRequest":
		logger.Debug("ProjectRequest detected, waiting for namespace creation...")
		fn, err := AssertNamespaceFn(ctx, m.logger, m.kube, r.Name)
		if err != nil {
			return err
		}
		m.queue = append(m.queue, fn)
	}
	return nil
}

// Watch waits for all monitoring functions to complete, or until the timeout is
// reached. Returns error if the queue is not empty after timeout.
func (m *Monitor) Watch(timeout time.Duration) error {
	start := time.Now()
	logger := m.logger.With(
		"timeout", timeout.String(),
		"start", start.Format(time.RFC3339),
		"queue-size", len(m.queue),
	)
	// Going through the queue of monitor functions, the successful items are
	// removed from the queue leaving only the functions which are returning
	// error.
	for len(m.queue) > 0 {
		// If the timeout is reached, return an error.
		if time.Since(start) >= timeout {
			return errors.New("timeout reached")
		}

		// Run the monitor function, if successful remove it from the queue.
		if err := m.queue[0](); err == nil {
			logger.Debug("Monitor function succeeded!",
				"queue-remaining", len(m.queue))
			m.queue = m.queue[1:]
		} else {
			logger.Debug("Monitor function failed!",
				"queue-remaining", len(m.queue))
			time.Sleep(2 * time.Second)
		}
	}
	logger.Debug("Monitoring complete, queue is empty!")
	return nil
}

// NewMonitor instantiates a new Monitor.
func NewMonitor(logger *slog.Logger, kube k8s.Interface) *Monitor {
	return &Monitor{
		logger: logger.With("type", "monitor"),
		kube:   kube,
		queue:  []monitorQueueFn{},
	}
}
