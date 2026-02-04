package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GetSecret retrieves a Kubernetes secret by full name.
func GetSecret(
	ctx context.Context,
	kube *Kube,
	name types.NamespacedName,
) (*corev1.Secret, error) {
	coreClient, err := kube.CoreV1ClientSet(name.Namespace)
	if err != nil {
		return nil, err
	}
	return coreClient.Secrets(name.Namespace).
		Get(ctx, name.Name, metav1.GetOptions{})
}

// SecretExists checks if a Kubernetes secret exists.
func SecretExists(
	ctx context.Context,
	kube *Kube,
	name types.NamespacedName,
) (bool, error) {
	_, err := GetSecret(ctx, kube, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteSecret deletes a Kubernetes secret.
func DeleteSecret(
	ctx context.Context,
	kube *Kube,
	name types.NamespacedName,
) error {
	coreClient, err := kube.CoreV1ClientSet(name.Namespace)
	if err != nil {
		return err
	}
	return coreClient.Secrets(name.Namespace).
		Delete(ctx, name.Name, metav1.DeleteOptions{})
}
