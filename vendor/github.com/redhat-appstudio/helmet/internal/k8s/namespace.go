package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureNamespace ensures the Kubernetes namespace exists.
// Uses vanilla Kubernetes Namespace API which works on both OpenShift and KinD.
func EnsureNamespace(
	ctx context.Context,
	logger *slog.Logger,
	kube Interface,
	namespace string,
) error {
	logger = logger.With("namespace", namespace)

	logger.Debug("Verifying Kubernetes client connection...")
	if err := kube.Connected(); err != nil {
		return err
	}

	client, err := kube.CoreV1ClientSet("default")
	if err != nil {
		return err
	}

	logger.Debug("Checking if namespace exists...")
	_, err = client.Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		logger.Debug("Namespace already exists.")
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	logger.Info("Creating namespace...")
	_, err = client.Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Wait for namespace to be ready
	logger.Info("Namespace created, waiting for it to be ready...")
	timeout := 30 * time.Second
	interval := 500 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for {
		ns, err := client.Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if ns.Status.Phase == corev1.NamespaceActive {
			logger.Info("Namespace is ready!")
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for namespace to be ready")
		}

		time.Sleep(interval)
	}
}
