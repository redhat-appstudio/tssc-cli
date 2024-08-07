package monitor

import (
	"context"
	"log/slog"
	"testing"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
	"github.com/redhat-appstudio/rhtap-cli/test/stubs"

	"k8s.io/apimachinery/pkg/runtime"
)

// TestAssertNamespaceFn tests the AssertNamespaceFn function, which returns error
// when the namespace informed is not found.
func TestAssertNamespaceFn(t *testing.T) {
	tests := []struct {
		name      string
		objects   []runtime.Object
		namespace string
		wantErr   bool
	}{{
		name:      "namespace not-found",
		objects:   []runtime.Object{},
		namespace: "default",
		wantErr:   true,
	}, {
		name:      "namespace exists",
		objects:   []runtime.Object{stubs.NamespaceRuntimeObject("test")},
		namespace: "test",
		wantErr:   false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := k8s.NewFakeKube(tt.objects...)
			fn, err := AssertNamespaceFn(
				context.TODO(),
				slog.Default(),
				kube,
				tt.namespace,
			)
			if err != nil {
				t.Errorf("AssertNamespaceFn() error = %v", err)
				return
			}
			if err = fn(); (err != nil) != tt.wantErr {
				t.Errorf("AssertNamespaceFn()->fn() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
		})
	}
}
