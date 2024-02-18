/*
Copyright 2024.

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

	danaiov1alpha1 "dana.io/namespacelabel/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)


// Create the NamespaceLabel Reconciler
type NamespaceLabelReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	ctrl.Manager
}

// List of protected labels that should not be modified by the NamespaceLabel CR
var protectedLabels = map[string]struct{}{
	"app.kubernetes.io/name":       {},
	"app.kubernetes.io/instance":   {},
	"app.kubernetes.io/version":    {},
	"app.kubernetes.io/component":  {},
	"app.kubernetes.io/part-of":    {},
	"app.kubernetes.io/managed-by": {},
}

// Check if a label is protected
func isProtected(label string) bool {
	_, ok := protectedLabels[label]
	return ok
}

func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_log := log.FromContext(ctx)

	// List all NamespaceLabel CRs for the namespace
	namespaceLabelList := &danaiov1alpha1.NamespaceLabelList{}
	if err := r.List(ctx, namespaceLabelList, client.InNamespace(req.Namespace)); err != nil {
		_log.Error(err, "unable to list NamespaceLabel CRs for namespace", "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// Sort NamespaceLabel CRs by creation timestamp to prioritize the latest
	sort.Slice(namespaceLabelList.Items, func(i, j int) bool {
		return namespaceLabelList.Items[i].CreationTimestamp.After(namespaceLabelList.Items[j].CreationTimestamp.Time)
	})

	// Merge labels from the most recent NamespaceLabel CR, overriding older labels
	finalLabels := make(map[string]string)
	if len(namespaceLabelList.Items) > 0 {
		latestCR := namespaceLabelList.Items[0]
		for key, value := range latestCR.Spec.Labels {
			finalLabels[key] = value
		}
	}

	// Fetch the target Namespace
	var namespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace); err != nil {
		return ctrl.Result{}, err
	}

	// Initialize namespace labels if nil
	if namespace.Labels == nil {
		namespace.Labels = map[string]string{}
	}
	// Remove all existing labels from the namespace if they are not protected
	for label := range namespace.Labels {
		if !isProtected(label) {
			delete(namespace.Labels, label)
		}
	}

	// Merge labels from the most recent NamespaceLabel CR, overriding older labels
	// Skip adding labels that are protected - from the list.
	if len(namespaceLabelList.Items) > 0 {
		latestCR := namespaceLabelList.Items[0]
		for key, value := range latestCR.Spec.Labels {
			if !isProtected(key) {
				namespace.Labels[key] = value
			}
		}
	}

	// Emit an event if the namespace labels are about to be overridden
	r.EventRecorder.Event(&namespace, corev1.EventTypeWarning, "Override", "This NamespaceLabel CR will override existing label configurations due to its more recent creation timestamp.")

	// Update the namespace labels
	if err := r.Update(ctx, &namespace); err != nil {
		_log.Error(err, "Failed to update Namespace labels", "Namespace", namespace.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Ensure the EventRecorder is initialized here or wherever appropriate
	if r.EventRecorder == nil {
		r.EventRecorder = mgr.GetEventRecorderFor("namespace-label-controller")
	}
	r.Manager = mgr

	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NamespaceLabel{}).
		Complete(r)
}
