package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	v1 "github.com/openshift/api/operator/v1"
	projectv1 "github.com/openshift/api/project/v1"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ErrIngressDomainNotFound returned when the OpenShift ingress domain is empty.
var ErrIngressDomainNotFound = fmt.Errorf("ingress domain not found")

// Returns `default` IngressController CR if exists.
func getIngressControllerCR(ctx context.Context, kube *Kube) (*v1.IngressController, error) {
	objectRef := &corev1.ObjectReference{
		APIVersion: "operator.openshift.io/v1",
		Namespace:  "openshift-ingress-operator",
		Name:       "default",
	}

	restConfig, err := kube.RESTClientGetter(objectRef.Namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	operatorClient, err := operatorv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	ingressController, err := operatorClient.
		IngressControllers(objectRef.Namespace).
		Get(ctx, objectRef.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrIngressDomainNotFound
		}
		return nil, err
	}
	return ingressController, nil
}

// Returns `version` ClusterVersion CR if exists.
func getConfigVersionCR(ctx context.Context, kube *Kube) (*configv1.ClusterVersion, error) {
	objectRef := &corev1.ObjectReference{
		APIVersion: "config.openshift.io/v1",
		Namespace:  "",
		Name:       "version",
	}

	restConfig, err := kube.RESTClientGetter(objectRef.Namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	configClient, err := configv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	version, err := configClient.
		ClusterVersions().
		Get(ctx, objectRef.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("cluster version not found")
		}
		return nil, err
	}
	return version, nil
}

// Returns name of the defaultCertificate as specified in default IngressController
func getIngressControllerDefaultCertificate(ctx context.Context, kube *Kube) (string, error) {
	ingressController, err := getIngressControllerCR(ctx, kube)
	if err != nil {
		return "", err
	}
	if ingressController.Spec.DefaultCertificate == nil {
		return "", nil
	}
	ingressCertSecretName := ingressController.Spec.DefaultCertificate.Name

	return ingressCertSecretName, nil
}

// GetOpenShiftIngressRouteCA returns base64-encoded root certificate for openshift-ingress route.
// Uses either what's defines in spec->defaultCertificate of IngressController or if that's not defined
// uses `router-ca` secret from `openshift-ingress-operator` namespace.
// Related documentation: https://docs.openshift.com/container-platform/4.18/security/certificates/replacing-default-ingress-certificate.html#replacing-default-ingress
func GetOpenShiftIngressRouteCA(ctx context.Context, kube *Kube) (string, error) {
	defaultCertSecretName, err := getIngressControllerDefaultCertificate(ctx, kube)
	if err != nil {
		return "", err
	}
	secretNamespacedName := types.NamespacedName{
		Namespace: "openshift-ingress-operator",
		Name:      "router-ca",
	}
	if defaultCertSecretName != "" { // if defaultCertificate is specified, use that instead
		secretNamespacedName = types.NamespacedName{
			Namespace: "openshift-ingress",
			Name:      defaultCertSecretName,
		}
	}
	secret, err := GetSecret(ctx, kube, secretNamespacedName)
	if err != nil {
		return "", err
	}

	certData, ok := secret.Data["tls.crt"]
	if !ok {
		return "", fmt.Errorf("tls.crt key not found in router-ca secret")
	}
	return base64.StdEncoding.EncodeToString(certData), nil
}

// GetOpenShiftIngressDomain returns the OpenShift Ingress domain.
func GetOpenShiftIngressDomain(ctx context.Context, kube *Kube) (string, error) {
	ingressController, err := getIngressControllerCR(ctx, kube)
	if err != nil {
		return "", err
	}

	ingressDomain := ingressController.Status.Domain
	if ingressDomain == "" {
		return "", ErrIngressDomainNotFound
	}
	return ingressDomain, nil
}

// GetOpenShiftVersion returns the OpenShift version.
func GetOpenShiftVersion(ctx context.Context, kube *Kube) (string, error) {
	clusterVersion, err := getConfigVersionCR(ctx, kube)
	if err != nil {
		return "", err
	}

	version := clusterVersion.Status.Desired.Version
	if version == "" {
		return "", fmt.Errorf("cluster desired version not found")
	}
	return version, nil
}

// EnsureOpenShiftProject ensures the OpenShift project exists.
func EnsureOpenShiftProject(
	ctx context.Context,
	logger *slog.Logger,
	kube *Kube,
	projectName string,
) error {
	logger = logger.With("project", projectName)

	logger.Debug("Verifying Kubernetes client connection...")
	if err := kube.Connected(); err != nil {
		return err
	}

	restConfig, err := kube.RESTClientGetter("default").ToRESTConfig()
	if err != nil {
		return err
	}
	projectClient, err := projectv1client.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	logger.Debug("ensuring project exists.")
	_, err = projectClient.Projects().Get(ctx, projectName, metav1.GetOptions{})
	if err == nil {
		logger.Debug("Project already exists.")
		return nil
	}

	projectRequest := &projectv1.ProjectRequest{
		DisplayName: projectName,
		Description: fmt.Sprintf("RHTAP: %s", projectName),
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
	}

	logger.Info("Creating OpenShift project...")
	_, err = projectClient.ProjectRequests().
		Create(ctx, projectRequest, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Grace time to ensure the namespace is ready
	logger.Info("OpenShift project created!")
	time.Sleep(5 * time.Second)
	return nil
}
