package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
	"github.com/redhat-appstudio/rhtap-cli/test/stubs"

	"k8s.io/cli-runtime/pkg/resource"

	o "github.com/onsi/gomega"
)

// TestMonitorCollect tests the Monitor's Collect function, which adds a
// monitoring function when a relevant resource is found.
func TestMonitorCollect(t *testing.T) {
	tests := []struct {
		name         string
		resourceInfo *resource.Info
		queueLength  int
		wantErr      bool
	}{{
		name:         "non-interesting resource",
		resourceInfo: stubs.PodResourceInfo("default", "test"),
		queueLength:  0,
		wantErr:      false,
	}, {
		name:         "ProjectRequest resource",
		resourceInfo: stubs.ProjectRequestResourceInfo("default", "test"),
		queueLength:  0,
		wantErr:      false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := k8s.NewFakeKube()
			m := NewMonitor(slog.Default(), kube)

			err := m.Collect(context.TODO(), tt.resourceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Monitor.Collect() error = %v, wantErr %v",
					err, tt.wantErr)
			}
			if len(m.queue) != tt.queueLength {
				t.Errorf("Monitor.Collect() queue length = %v, want %v",
					len(m.queue), tt.queueLength)
			}
		})
	}
}

// TestMonitorWatch tests the Monitor's Watch function, which waits for all
// monitoring functions to complete or until the timeout is reached.
func TestMonitorWatch(t *testing.T) {
	g := o.NewWithT(t)

	noopFn := func() error { return nil }
	oneSecondSleepFn := func() error {
		time.Sleep(1 * time.Second)
		return fmt.Errorf("generic error")
	}

	t.Run("Timeout", func(t *testing.T) {
		m := &Monitor{
			logger: slog.Default(),
			kube:   k8s.NewFakeKube(),
			queue:  []monitorQueueFn{noopFn, oneSecondSleepFn},
		}
		err := m.Watch(500 * time.Millisecond)
		g.Expect(err).To(o.HaveOccurred())
	})

	t.Run("Success", func(t *testing.T) {
		m := &Monitor{
			logger: slog.Default(),
			kube:   k8s.NewFakeKube(),
			queue:  []monitorQueueFn{noopFn, noopFn, noopFn},
		}
		err := m.Watch(500 * time.Millisecond)
		g.Expect(err).ToNot(o.HaveOccurred())
	})
}
