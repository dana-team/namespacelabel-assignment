package controller_test

import (
	"context"
	"testing"
	"time"

	danaiov1alpha1 "dana.io/namespacelabel/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	controller "dana.io/namespacelabel/internal/controller" // Adjust the import path as necessary
)

func TestNamespaceLabelReconciler_Reconcile(t *testing.T) {
	// Scheme includes Kubernetes and NamespaceLabel API groups
	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	_ = danaiov1alpha1.AddToScheme(sch)

	// Prepare test namespace and NamespaceLabel instances
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	testNamespaceLabel := &danaiov1alpha1.NamespaceLabel{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-namespacelabel",
			Namespace:         "test-namespace",
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: danaiov1alpha1.NamespaceLabelSpec{
			Labels: map[string]string{
				"testlabel": "testvalue",
			},
		},
	}

	// Initialize the fake client with the test objects
	cl := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(testNamespace, testNamespaceLabel).Build()

	// Create an instance of the Reconciler
	r := &controller.NamespaceLabelReconciler{
		Client: cl,
		Scheme: sch,
		// Mocking the EventRecorder for testing purposes
		EventRecorder: &FakeRecorder{},
	}

	// Create a reconcile request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-namespace",
			Namespace: "test-namespace",
		},
	}

	// Invoke the Reconcile method
	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Fetch the updated namespace to check if labels were updated correctly
	var updatedNamespace corev1.Namespace
	err = cl.Get(context.Background(), client.ObjectKey{Name: "test-namespace"}, &updatedNamespace)
	if err != nil {
		t.Fatalf("get namespace: (%v)", err)
	}

	// Validate the namespace labels
	expectedLabel := "testlabel"
	if val, ok := updatedNamespace.Labels[expectedLabel]; !ok || val != "testvalue" {
		t.Errorf("expected label %s to be 'testvalue', got '%s'", expectedLabel, val)
	}
}

// FakeRecorder is a fake record.EventRecorder for testing
type FakeRecorder struct{}

func (*FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {}
func (*FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}
func (*FakeRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
