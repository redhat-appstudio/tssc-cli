package stubs

import (
	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
)

// PodResourceInfo returns a resource.Info with a Pod object.
func PodResourceInfo(namespace, name string) *resource.Info {
	return &resource.Info{
		Namespace: namespace,
		Name:      name,
		Object: runtime.Object(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		}),
	}
}

// ProjectRequestResourceInfo returns a resource.Info with a ProjectRequest.
func ProjectRequestResourceInfo(namespace, name string) *resource.Info {
	return &resource.Info{
		Namespace: namespace,
		Name:      name,
		Object: runtime.Object(&projectv1.ProjectRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			DisplayName: "project-request",
			Description: "project-request",
		}),
	}
}
