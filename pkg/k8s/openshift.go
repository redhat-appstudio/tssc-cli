package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	projectv1 "github.com/openshift/api/project/v1"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ErrIngressDomainNotFound returned when the OpenShift ingress domain is empty.
var ErrIngressDomainNotFound = fmt.Errorf("ingress domain not found")

// GetOpenShiftIngressDomain returns the OpenShift Ingress domain.
func GetOpenShiftIngressDomain(ctx context.Context, kube *Kube) (string, error) {
	objectRef := &corev1.ObjectReference{
		APIVersion: "operator.openshift.io/v1",
		Namespace:  "openshift-ingress-operator",
		Name:       "default",
	}

	restConfig, err := kube.RESTClientGetter(objectRef.Namespace).ToRESTConfig()
	if err != nil {
		return "", err
	}
	operatorClient, err := operatorv1client.NewForConfig(restConfig)
	if err != nil {
		return "", err
	}

	ingressController, err := operatorClient.
		IngressControllers(objectRef.Namespace).
		Get(ctx, objectRef.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", ErrIngressDomainNotFound
		}
		return "", err
	}

	ingressDomain := ingressController.Status.Domain
	if ingressDomain == "" {
		return "", ErrIngressDomainNotFound
	}
	return ingressDomain, nil
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
