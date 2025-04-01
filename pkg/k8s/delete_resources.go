package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// labelSelector is the label set for resources to be deleted.
const labelSelector = "rhtap-cli.redhat-appstudio.github.com/post-deploy=delete"

// DeleteClusterRoleBindings deletes Kubernetes ClusterRoleBindings by label.
func DeleteClusterRoleBindings(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	rbacClient, err := kube.RBACV1ClientSet(namespace)
	if err != nil {
		return err
	}
	return rbacClient.ClusterRoleBindings().
		DeleteCollection(ctx, metav1.DeleteOptions{},
			metav1.ListOptions{LabelSelector: labelSelector})
}

// DeleteClusterRoles deletes Kubernetes ClusterRoles by label.
func DeleteClusterRoles(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	rbacClient, err := kube.RBACV1ClientSet(namespace)
	if err != nil {
		return err
	}
	return rbacClient.ClusterRoles().
		DeleteCollection(ctx, metav1.DeleteOptions{},
			metav1.ListOptions{LabelSelector: labelSelector})
}

// DeleteRoleBindings deletes Kuberbetes RoleBindings by label.
func DeleteRoleBindings(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	rbacClient, err := kube.RBACV1ClientSet(namespace)
	if err != nil {
		return err
	}
	RoleBindingsList, err := rbacClient.RoleBindings("").
		List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	for _, rb := range RoleBindingsList.Items {
		err := rbacClient.RoleBindings(rb.Namespace).
			Delete(ctx, rb.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteRoles deletes Kubernetes Roles by label.
func DeleteRoles(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	rbacClient, err := kube.RBACV1ClientSet(namespace)
	if err != nil {
		return err
	}
	RolesList, err := rbacClient.Roles("").
		List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	for _, role := range RolesList.Items {
		err := rbacClient.Roles(role.Namespace).
			Delete(ctx, role.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteServiceAccounts deletes Kubernetes ServiceAccounts by label.
func DeleteServiceAccounts(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	coreClient, err := kube.CoreV1ClientSet(namespace)
	if err != nil {
		return err
	}
	ServiceAccountList, err := coreClient.ServiceAccounts("").
		List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	for _, sa := range ServiceAccountList.Items {
		err := coreClient.ServiceAccounts(sa.Namespace).
			Delete(ctx, sa.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteResources deletes temporary Kubernetes resources created during deployment.
func DeleteResources(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	// Delete ClusterRoleBindings
	if err := DeleteClusterRoleBindings(ctx, kube, namespace); err != nil {
		return err
	}
	// Delete ClusterRoles
	if err := DeleteClusterRoles(ctx, kube, namespace); err != nil {
		return err
	}
	// Delete RoleBindings
	if err := DeleteRoleBindings(ctx, kube, namespace); err != nil {
		return err
	}
	// Delete Roles
	if err := DeleteRoles(ctx, kube, namespace); err != nil {
		return err
	}
	// Delete ServiceAccounts
	if err := DeleteServiceAccounts(ctx, kube, namespace); err != nil {
		return err
	}
	return nil
}

// Retry defines retry.
func Retry(attempts int, sleep time.Duration, fn func() error) error {
	for i := 0; ; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		if i >= (attempts - 1) {
			return err
		}
		time.Sleep(sleep)
	}
}

// RetryDeleteResources deletes temporary resources with retry 5 times.
func RetryDeleteResources(
	ctx context.Context,
	kube *Kube,
	namespace string,
) error {
	// Delete temporary resources
	err := Retry(5, 10*time.Second, func() error {
		err := DeleteResources(ctx, kube, namespace)
		return err
	})
	return err
}
