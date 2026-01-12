/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	namespacelabelv1alpha1 "namespacelabel.dana.io/api/v1alpha1"
)

const (
	// Annotation used to track which labels this operator is responsible for.
	ManagedLabelsAnnotation = "namespacelabel.dana.io/managed-labels"
)

// List of prefixes that we should NEVER touch.
var protectedPrefixes = []string{"kubernetes.io/", "k8s.io/"}

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=namespacelabel.dana.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=namespacelabel.dana.io,resources=namespacelabels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=namespacelabel.dana.io,resources=namespacelabels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceLabel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Fetch all NamespaceLabel objects in the namespace
	var nlList namespacelabelv1alpha1.NamespaceLabelList
	if err := r.List(ctx, &nlList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the Namespace object
	var ns corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Determine desired labels
	desiredLabels := r.calculateDesiredLabels(nlList.Items)

	// Get currently managed labels from annotation
	managedLabelsStr := ns.Annotations[ManagedLabelsAnnotation]
	managedLabelsKeys := make(map[string]struct{})
	if managedLabelsStr != "" {
		for _, k := range strings.Split(managedLabelsStr, ",") {
			managedLabelsKeys[k] = struct{}{}
		}
	}

	// Update Namespace labels
	changed := false
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}

	// 1. Remove labels that are no longer managed
	for k := range managedLabelsKeys {
		if _, ok := desiredLabels[k]; !ok {
			// Only remove if it's not protected
			if !r.isProtected(k) {
				delete(ns.Labels, k)
				changed = true
			}
		}
	}

	// 2. Add/Update desired labels
	newManagedKeys := []string{}
	for k, v := range desiredLabels {
		if !r.isProtected(k) {
			if ns.Labels[k] != v {
				ns.Labels[k] = v
				changed = true
			}
			newManagedKeys = append(newManagedKeys, k)
		}
	}

	// Update annotation
	sort.Strings(newManagedKeys)
	newManagedLabelsStr := strings.Join(newManagedKeys, ",")
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	if ns.Annotations[ManagedLabelsAnnotation] != newManagedLabelsStr {
		ns.Annotations[ManagedLabelsAnnotation] = newManagedLabelsStr
		changed = true
	}

	if changed {
		if err := r.Update(ctx, &ns); err != nil {
			l.Error(err, "unable to update Namespace labels")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceLabelReconciler) calculateDesiredLabels(items []namespacelabelv1alpha1.NamespaceLabel) map[string]string {
	// Sort items by name in descending order (e.g., "z", "b", "a").
	// Since "a" comes last in a descending sort, its labels will overwrite
	// any labels set by "b" or "z" when we iterate through the list.
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name > items[j].Name
	})

	desired := make(map[string]string)
	for _, item := range items {
		for k, v := range item.Spec.Labels {
			desired[k] = v
		}
	}
	return desired
}
func (r *NamespaceLabelReconciler) isProtected(key string) bool {
	for _, p := range protectedPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&namespacelabelv1alpha1.NamespaceLabel{}).
		Named("namespacelabel").
		Complete(r)
}
