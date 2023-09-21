/*
Copyright 2023.

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

	labelsv1alpha1 "github.com/dvirgilad/namespacelabel-assignment/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	log "sigs.k8s.io/controller-runtime/pkg/log"
)

// CustomLabelsReconciler reconciles a CustomLabels object
type CustomLabelsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=labels.my.domain,resources=customlabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=labels.my.domain,resources=customlabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=labels.my.domain,resources=customlabels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CustomLabels object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *CustomLabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var customLabels = &labelsv1alpha1.CustomLabels{}
	if err := r.Get(ctx, req.NamespacedName, customLabels); err != nil {
		log.Error(err, "unable to fetch custom labels")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, namespace)
	if err != nil {
		log.Error(err, "unable to find Namespace")
		customLabels.Status.Applied = false
		return ctrl.Result{}, err
	}
	labelsToAdd := customLabels.Spec.CustomLabels

	deleteLabelsFinalizer := "labels.my.domain/finalizer"

	// Delete labels if object is deleted

	if !customLabels.ObjectMeta.DeletionTimestamp.IsZero() {
		//object is being deleted
		for k := range labelsToAdd {
			delete(namespace.Labels, k)
		}

		if err := r.Update(ctx, namespace); err != nil {
			log.Error(err, "unable to remove namespace labels")
			return ctrl.Result{}, err
		}
		// delete finalizer
		controllerutil.RemoveFinalizer(customLabels, deleteLabelsFinalizer)
		if err := r.Update(ctx, customLabels); err != nil {
			log.Error(err, "unable to delete finalizer")
			return ctrl.Result{}, err
		}
		log.Info("deleted namespace labels")
		return ctrl.Result{}, nil

	} else {
		// object is not being deleted - add finalizer
		if !controllerutil.ContainsFinalizer(customLabels, deleteLabelsFinalizer) {
			controllerutil.AddFinalizer(customLabels, deleteLabelsFinalizer)
			if err := r.Update(ctx, customLabels); err != nil {
				log.Error(err, "unable to add finalizer")
				return ctrl.Result{}, err
			}
			log.Info("added finalizer")
		} else {
			log.Info("finalizer already present")
		}
	}

	for k, v := range labelsToAdd {
		namespace.Labels[k] = v
	}
	err = r.Update(ctx, namespace)
	if err != nil {
		log.Error(err, "failed to update namespace")
		customLabels.Status.Applied = false
		return ctrl.Result{}, err
	}
	log.Info("added namespace labels")
	customLabels.Status.Applied = true
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomLabelsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1alpha1.CustomLabels{}).
		Complete(r)
}
