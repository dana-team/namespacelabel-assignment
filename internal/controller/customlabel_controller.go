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
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"os"
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
func (r *CustomLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
	r.Log = zap.New(core, zap.AddCaller())
	log := r.Log
	protectedPrefixArray := strings.Split(r.ProtectedPrefixes, ",")
	var customLabels = &labelsv1.CustomLabel{}
	if err := r.Get(ctx, req.NamespacedName, customLabels); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error("unable to fetch custom labels", zap.Error(err))
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}

	}
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, namespace)
	if err != nil {
		log.Error("unable to find Namespace", zap.Error(err))
		r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())
		return ctrl.Result{}, err
	}

	if customLabels.ObjectMeta.DeletionTimestamp.IsZero() {
		//object is not being deleted
		//add finalizer
		ok, err := r.AddFinalizer(ctx, customLabels, log)
		if err != nil {
			r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())

			return ctrl.Result{}, err
		}
		if ok {

			return ctrl.Result{}, nil
		}

	} else {
		// object is being deleted
		log.Info("deleting labels")
		//check if deleting protected labels and delete labels
		r.DeleteNameSpaceLabels(customLabels, namespace)
		// remove labels from namespace
		if err := r.Client.Update(ctx, namespace); err != nil {
			r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())

			return ctrl.Result{}, err

		}
		log.Info("deleted labels from namespace")
		ok, err := r.DeleteFinalizer(ctx, customLabels, log)
		if err != nil {
			r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())

			return ctrl.Result{}, err
		} else {
			if ok {
				return ctrl.Result{}, nil
			}
		}

	}
	// delete old labels
	log.Info("deleting stale labels")
	r.DeleteNameSpaceLabels(customLabels, namespace)

	//add labels
	if err := r.AddNamespaceLabels(customLabels, namespace, protectedPrefixArray); err != nil {
		log.Error("unable to add labels", zap.Error(err))
		r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())
		return ctrl.Result{}, fmt.Errorf("failed to add labels: %s", err.Error())
	}
	if err := r.Client.Update(ctx, namespace); err != nil {
		log.Error("error adding labels", zap.Error(err))
		customLabels.Status.Applied = false
		customLabels.Status.Message = "error adding labels to namespace"
		if err := r.Client.Status().Update(ctx, customLabels); err != nil {
			log.Error("unable to modify custom label status", zap.Error(err))
			r.UpdateCustomLabelStatus(ctx, customLabels, false, err.Error())

			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	log.Info("edited namespace with new labels")

	log.Info("updating labels object status")
	r.UpdateCustomLabelStatus(ctx, customLabels, true, "labels applied")

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
