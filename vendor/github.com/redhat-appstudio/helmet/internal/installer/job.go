package installer

import (
	"context"
	"fmt"
	"slices"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	applyrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
)

// Job represents the asynchronous actor that runs a Job in the cluster to run
// this installer container image on a pod. The idea is to allow a non-blocking
// installation process for the MCP server.
type Job struct {
	kube    *k8s.Kube // kubernetes client
	appName string    // common name for resources
	retries int32     // job retries
}

// LabelSelector returns the label selector for installer jobs.
func (j *Job) LabelSelector() string {
	return fmt.Sprintf("installer-job.%s", annotations.RepoURI)
}

// JobState represents the state of the installer job in the cluster.
type JobState int

const (
	// NotFound no installer job found in the cluster.
	NotFound JobState = iota
	// Deploying the installer job is running.
	Deploying
	// Failed the installer job has failed.
	Failed
	// Done the installer job has succeeded.
	Done
)

// getJob retrieves the current state of the installer job. When not found it
// returns a nil job.
func (j *Job) getJob(ctx context.Context) (*batchv1.Job, error) {
	bc, err := j.kube.BatchV1ClientSet("")
	if err != nil {
		return nil, err
	}

	jobList, err := bc.Jobs("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("type=%s", j.LabelSelector()),
	})
	if err != nil {
		return nil, err
	}

	// When the job list is empty, it returns a nil job as well.
	if len(jobList.Items) == 0 {
		return nil, nil
	}

	// Returns error when multiple installer jobs are found in the cluster, with
	// information to identify them.
	if len(jobList.Items) > 1 {
		jobs := []string{}
		for _, job := range jobList.Items {
			jobs = append(jobs, fmt.Sprintf(
				"%s/%s", job.GetNamespace(), job.GetName(),
			))
		}
		return nil, fmt.Errorf("multiple installer jobs found: %v", jobs)
	}

	return &jobList.Items[0], nil
}

// GetState retrieves the current state of the installation job.
func (j *Job) GetState(ctx context.Context) (JobState, error) {
	job, err := j.getJob(ctx)
	if err != nil {
		return -1, err
	}

	if job == nil {
		return NotFound, nil
	}
	// Checking whether the existing job is a dry-run container, in this case the
	// overall installer state is considered "not found".
	podSpec := job.Spec.Template.Spec
	if len(podSpec.Containers) == 1 {
		container := podSpec.Containers[0]
		if slices.Contains(container.Args, "--dry-run") {
			return NotFound, nil
		}
	}

	if job.Status.Active > 0 {
		return Deploying, nil
	}
	if job.Status.Failed > 0 {
		return Failed, nil
	}
	if job.Status.Succeeded > 0 {
		return Done, nil
	}
	return -1, fmt.Errorf("unknown job state")
}

// applyServiceAccount applies a ServiceAccount to the cluster.
func (j *Job) applyServiceAccount(ctx context.Context, namespace string) error {
	cc, err := j.kube.CoreV1ClientSet("")
	if err != nil {
		return err
	}

	apiVersion := "v1"
	kind := "ServiceAccount"
	sa := &applycorev1.ServiceAccountApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			APIVersion: &apiVersion,
			Kind:       &kind,
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Namespace: &namespace,
			Name:      &j.appName,
		},
	}
	_, err = cc.ServiceAccounts(namespace).Apply(ctx, sa, metav1.ApplyOptions{
		FieldManager: j.appName,
	})
	return err
}

// applyClusterRoleBinding applies a ClusterRoleBinding to the ServiceAccount.
func (j *Job) applyClusterRoleBinding(
	ctx context.Context,
	namespace string,
) error {
	rc, err := j.kube.RBACV1ClientSet("")
	if err != nil {
		return err
	}

	roleRefAPIGroup := rc.RESTClient().APIVersion().Group
	roleRefKind := "ClusterRole"
	roleRefName := "cluster-admin"
	subjectKind := "ServiceAccount"

	apiVersion := "rbac.authorization.k8s.io/v1"
	kind := "ClusterRoleBinding"

	crb := &applyrbacv1.ClusterRoleBindingApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			APIVersion: &apiVersion,
			Kind:       &kind,
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: &j.appName,
		},
		RoleRef: &applyrbacv1.RoleRefApplyConfiguration{
			APIGroup: &roleRefAPIGroup,
			Kind:     &roleRefKind,
			Name:     &roleRefName,
		},
		Subjects: []applyrbacv1.SubjectApplyConfiguration{{
			Kind:      &subjectKind,
			Namespace: &namespace,
			Name:      &j.appName,
		}},
	}
	_, err = rc.ClusterRoleBindings().Apply(ctx, crb, metav1.ApplyOptions{
		FieldManager: j.appName,
	})
	return err
}

// createJob creates a Kubernetes Job to deploy TSSC, preparing the installer to
// run on a container image and connect to the Kubernetes API in-cluster.
func (j *Job) createJob(
	ctx context.Context,
	debug, dryRun bool,
	namespace, image string,
) error {
	bc, err := j.kube.BatchV1ClientSet("")
	if err != nil {
		return err
	}

	// Setting up the list of arguments for the deployment job.
	args := []string{"deploy"}
	if debug {
		args = append(args, "--debug")
		args = append(args, "--log-level=debug")
	}
	if dryRun {
		args = append(args, "--dry-run")
	}

	podSpec := corev1.PodSpec{
		ServiceAccountName: j.appName,
		Containers: []corev1.Container{{
			Name:  fmt.Sprintf("%s-deploy", j.appName),
			Image: image,
			Env: []corev1.EnvVar{{
				// KUBECONFIG must be empty to indicate that the job is running in
				// the cluster, using the service account credentials.
				Name:  "KUBECONFIG",
				Value: "",
			}},
			Args: args,
		}},
		RestartPolicy: corev1.RestartPolicyNever,
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-deploy-job", j.appName),
			Labels:    map[string]string{"type": j.LabelSelector()},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"type": j.LabelSelector()},
				},
				Spec: podSpec,
			},
			BackoffLimit: &j.retries,
		},
	}
	_, err = bc.Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}

// deleteJob deletes the installer job.
func (j *Job) deleteJob(ctx context.Context) error {
	bc, err := j.kube.BatchV1ClientSet("")
	if err != nil {
		return err
	}

	job, err := j.getJob(ctx)
	if err != nil {
		return err
	}
	return bc.Jobs(job.GetNamespace()).
		Delete(ctx, job.GetName(), metav1.DeleteOptions{})
}

// GetJobLogFollowCmd returns the command that follows the deployment job logs.
func (j *Job) GetJobLogFollowCmd(namespace string) string {
	return fmt.Sprintf(
		"oc --namespace=%s logs --follow --selector=\"type=%s\"",
		namespace,
		j.LabelSelector(),
	)
}

// Run issues a new installation job, creating the installation job when
// applicable. It applies the service account and cluster role binding first, then
// creates the job.
func (j *Job) Run(
	ctx context.Context,
	debug, dryRun, force bool,
	namespace, image string,
) error {
	state, err := j.GetState(ctx)
	if err != nil {
		return err
	}
	// The deployment job can only be created once, per cluster.
	if state != NotFound {
		// Upon force flag, the job is deleted before recreation.
		if force {
			if err = j.deleteJob(ctx); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("only a single deployment job is allowed")
		}
	}

	// Issuing the service account and cluster role binding first, the job needs
	// to run as cluster admin.
	if err = j.applyServiceAccount(ctx, namespace); err != nil {
		return fmt.Errorf("unable to apply the service account: %s", err)
	}
	if err = j.applyClusterRoleBinding(ctx, namespace); err != nil {
		return fmt.Errorf("unable to apply the cluster role binding: %s", err)
	}
	// Creating the job itself.
	return j.createJob(ctx, debug, dryRun, namespace, image)
}

// NewJob instantiates a new Job object.
func NewJob(appCtx *api.AppContext, kube *k8s.Kube) *Job {
	return &Job{
		kube:    kube,
		appName: appCtx.Name,
		retries: 0,
	}
}
