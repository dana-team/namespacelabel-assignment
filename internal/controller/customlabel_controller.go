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
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CustomLabelReconciler reconciles a CustomLabel object
type CustomLabelReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ProtectedPrefixes string
	Log               *zap.Logger
}

const DeleteLabelsFinalizer = "labels.dvir.io/finalizer"

// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=labels.dvir.io,resources=customlabels/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.v1,resources=namespace,verbs=watch;update

func (r *CustomLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log
	var customLabels = &labelsv1.CustomLabel{}
	if err := r.Get(ctx, req.NamespacedName, customLabels); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(fmt.Sprintf("unable to fetch custom labels: %s", req.Name), zap.Error(err))
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}

	}
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, namespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to find Namespace: %s", req.Namespace), zap.Error(err))
		if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error(), map[string]labelsv1.LabelStatus{}); err != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	if !customLabels.ObjectMeta.DeletionTimestamp.IsZero() {
		// object is being deleted
		log.Info("deleting labels")
		return r.HandleDelete(ctx, customLabels, namespace)
	}
	//object is not being deleted
	ok, err := r.AddFinalizer(ctx, customLabels, log)
	if err != nil {
		if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error(), map[string]labelsv1.LabelStatus{}); err != nil {
			return ctrl.Result{}, statusErr
		}

		return ctrl.Result{}, err
	}
	if ok {

		return ctrl.Result{}, nil
	}

	labelsToAdd := r.ParseLabels(customLabels, namespace)
	if len(labelsToAdd) == 0 {
		log.Info("no new labels to add")
		return ctrl.Result{}, nil
	}

	labelsStatus := r.AddNamespaceLabels(customLabels, namespace, strings.Split(r.ProtectedPrefixes, ","), labelsToAdd)
	if err := r.UpdateNamespace(ctx, customLabels, namespace); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("edited namespace with new labels")

	log.Info("updating labels object status")
	if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, true, "labels applied", labelsStatus); err != nil {
		return ctrl.Result{}, statusErr
	}

	log.Info("added namespace labels")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1.CustomLabel{}).Watches(
		&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(r.EnqueueRequestsOnNamespaceChange),
	).Complete(r)
}

func (r *CustomLabelReconciler) EnqueueRequestsOnNamespaceChange(ctx context.Context, object client.Object) []reconcile.Request {
	updatedNamespace := object.(*corev1.Namespace)
	customLabelList := &labelsv1.CustomLabelList{}
	if err := r.List(ctx, customLabelList, client.InNamespace(updatedNamespace.Name)); err != nil {
		/// can't get labels, return nothing
		return []reconcile.Request{}
	}
	var requests []reconcile.Request
	for _, customLabel := range customLabelList.Items {
		for k, v := range customLabel.Spec.CustomLabels {
			if updatedNamespace.Labels[k] != v {
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      customLabel.Name,
						Namespace: customLabel.Namespace,
					},
				}
				requests = append(requests, req)
			}
		}

	}
	if len(requests) == 0 {
		return []reconcile.Request{}
	}
	return requests
}
