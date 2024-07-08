package deployer

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	projectv1 "github.com/openshift/api/project/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func EnsureOpenShiftProject(
	logger *slog.Logger,
	kube *k8s.Kube,
	projectName string,
) error {
	logger = logger.With("project", projectName)

	restConfig, err := kube.RESTClientGetter("default").ToRESTConfig()
	if err != nil {
		return err
	}

	projectClient, err := projectv1client.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	ctx := backgroundContext(func() {
		logger.Warn("Project creation has been cancelled.")
	})

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
