package monitor

import (
	"context"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AssertNamespaceFn returns a function that asserts if the informed namespace
// exists, otherwise returns error.
func AssertNamespaceFn(
	ctx context.Context,
	logger *slog.Logger,
	kube k8s.Interface,
	namespace string,
) (monitorQueueFn, error) {
	client, err := kube.CoreV1ClientSet("default")
	if err != nil {
		return nil, err
	}
	return func() error {
		logger = logger.With("namespace", namespace)
		logger.Debug("Asserting namespace exists...")
		_, err := client.Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err == nil {
			logger.Debug("Namespace exists!")
		} else {
			logger.Debug("Namespace is not found!")
		}
		return err
	}, nil
}
