package controller

import (
	"context"
	"errors"
	"fmt"
	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

func (r *CustomLabelReconciler) AddFinalizer(ctx context.Context, customLabels *labelsv1.CustomLabel, log *zap.Logger) (ok bool, err error) {

	if !controllerutil.ContainsFinalizer(customLabels, DeleteLabelsFinalizer) {
		log.Info("adding finalizer")
		controllerutil.AddFinalizer(customLabels, DeleteLabelsFinalizer)
		if err := r.Update(ctx, customLabels); err != nil {
			log.Error("unable to add finalizer", zap.Error(err))
			return false, err
		}
		log.Info("added finalizer")
		return true, nil
	} else {

		return false, nil
	}

}

func (r *CustomLabelReconciler) DeleteFinalizer(ctx context.Context, customLabels *labelsv1.CustomLabel, log *zap.Logger) (bool, error) {
	if controllerutil.ContainsFinalizer(customLabels, DeleteLabelsFinalizer) {
		log.Info("removing finalizer")
		// remove finalizer
		controllerutil.RemoveFinalizer(customLabels, DeleteLabelsFinalizer)
		if err := r.Update(ctx, customLabels); err != nil {
			log.Error("error removing finalizer", zap.Error(err))
			return false, err
		}
		log.Info("Removed finalizer")
		return true, nil
	} else {
		//Finalizer already deleted
		return false, nil
	}
}
func (r *CustomLabelReconciler) AddNamespaceLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace, protectedPrefixArray []string) error {
	for k, v := range customLabel.Spec.CustomLabels {
		// Skip protected labels that contain a protected prefix
		for _, j := range protectedPrefixArray {
			if strings.Contains(k, j) {
				err := errors.New("attempting to add a label containing protected string")
				return err
			}
		}
		_, ok := namespace.Labels[k]
		if ok {
			err := errors.New(fmt.Sprintf("attempting to edit a label controlled by another crd: %s", k))
			return err
		}
		// Add label to namespace
		namespace.Labels[k] = v
	}
	return nil
}

func (r *CustomLabelReconciler) DeleteNameSpaceLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace) {
	for k, v := range namespace.ObjectMeta.Labels {
		_, ok := customLabel.Spec.CustomLabels[k]
		if ok && v == customLabel.Spec.CustomLabels[k] {
			// Delete labels with that exist in the CRD and that have the same value
			delete(namespace.Labels, k)
		}
	}
}

// Updates the status of the CRD with any errors that occured or if it succeeded
func (r *CustomLabelReconciler) UpdateCustomLabelStatus(ctx context.Context, CustomLabel *labelsv1.CustomLabel, applied bool, message string) {
	CustomLabel.Status.Applied = applied
	CustomLabel.Status.Message = message
	if err := r.Client.Status().Update(ctx, CustomLabel); err != nil {
		r.Log.Error("unable to modify custom label status", zap.Error(err))
	}
}
