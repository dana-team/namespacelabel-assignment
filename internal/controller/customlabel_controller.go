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
	"fmt"
	"strings"

	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CustomLabelReconciler reconciles a CustomLabel object
type CustomLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels/finalizers,verbs=update
func (r *CustomLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var customLabels = &labelsv1.CustomLabel{}
	if err := r.Get(ctx, req.NamespacedName, customLabels); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch custom labels")
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}

	}
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, namespace)
	if err != nil {
		log.Error(err, "unable to find Namespace")
		return ctrl.Result{}, err
	}
	labelsToAdd := customLabels.Spec.CustomLabels
	DeleteLabelsFinalizer := "labels.dvir.io/finalizer"

	if customLabels.ObjectMeta.DeletionTimestamp.IsZero() {
		//object is not being deleted
		//add finalizer
		if !controllerutil.ContainsFinalizer(customLabels, DeleteLabelsFinalizer) {
			log.Info("adding finalizer")
			controllerutil.AddFinalizer(customLabels, DeleteLabelsFinalizer)
			if err := r.Update(ctx, customLabels); err != nil {

				log.Error(err, "unable to add finalizer")
				return ctrl.Result{}, err
			}
			log.Info("added finalizer")
			return ctrl.Result{}, nil
		}

	} else {
		// object is being deleted
		log.Info("deleting labels")
		//check if deleting protected labels and delete labels
		if controllerutil.ContainsFinalizer(customLabels, DeleteLabelsFinalizer) {
			log.Info("removing finalizer")
			if err := r.deleteNameSpaceLabels(ctx, customLabels, namespace); err != nil {
				log.Error(err, "unable to remove labels")
				return ctrl.Result{}, err
			}
			log.Info("deleted labels from namespace")
			// remove finalizer
			controllerutil.RemoveFinalizer(customLabels, DeleteLabelsFinalizer)
			if err := r.Update(ctx, customLabels); err != nil {
				log.Error(err, "error removing finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Removed finalizer")
			return ctrl.Result{}, nil
		}

		log.Info("removed namespace labels")
		return ctrl.Result{}, nil
	}
	// delete old labels
	log.Info("deleting stale labels")
	for k := range namespace.ObjectMeta.Labels {
		// Skip protected labels that contain "kubernetes.io"
		if strings.Contains(k, "kubernetes.io") {
			continue
		}
		if strings.HasPrefix(k, customLabels.Name) {
			// Delete labels with prefix
			delete(namespace.Labels, k)
		}

	}
	for k, v := range labelsToAdd {
		// Skip protected labels that contain "kubernetes.io"
		if strings.Contains(k, "kubernetes.io") {
			continue
		}

		// Prefix the label key with the name of the custom resource
		customKey := fmt.Sprintf("%s/%s", customLabels.Name, k)
		namespace.Labels[customKey] = v
	}

	if err := r.Client.Update(ctx, namespace); err != nil {
		log.Error(err, "error adding labels")
		customLabels.Status.Applied = false
		customLabels.Status.Message = "error adding labels to namespace"
		if err := r.Client.Status().Update(ctx, customLabels); err != nil {
			log.Error(err, "unable to modify custom label status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	customLabels.Status.Applied = true
	customLabels.Status.Message = "applied namespace labels"
	log.Info("updating labels object status")

	if err := r.Client.Status().Update(ctx, customLabels); err != nil {
		log.Error(err, "unable to modify custom label status")

		return ctrl.Result{}, err
	}
	log.Info("added namespace labels")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1.CustomLabel{}).
		Complete(r)
}
func (r *CustomLabelReconciler) deleteNameSpaceLabels(ctx context.Context, customLabel *labelsv1.CustomLabel, NameSpace *corev1.Namespace) error {
	for k := range NameSpace.ObjectMeta.Labels {
		// Skip protected labels that contain "kubernetes.io"
		if strings.Contains(k, "kubernetes.io") {
			continue
		}

		// Prefix the label key with the name of the custom resource
		if strings.HasPrefix(k, customLabel.Name) {
			// Prefix the label key with the name of the custom resource
			delete(NameSpace.ObjectMeta.Labels, k)
		}
	}
	// remove labels from namespace
	if err := r.Client.Update(ctx, NameSpace); err != nil {
		return err
	}
	return nil
}
