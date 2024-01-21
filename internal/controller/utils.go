package controller

import (
	"context"
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
		contains := false
		for _, j := range protectedPrefixArray {
			if strings.Contains(k, j) {
				contains = true
				break
			}
		}
		if !contains {
			// Prefix the label key with the name of the custom resource
			customKey := fmt.Sprintf("%s/%s", customLabel.Name, k)
			namespace.Labels[customKey] = v
		}
	}
	return nil
}

func (r *CustomLabelReconciler) DeleteNameSpaceLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace, protectedPrefixArray []string) error {
	for k := range namespace.ObjectMeta.Labels {
		// Skip protected labels that contain a protected prefix
		contains := false
		for _, j := range protectedPrefixArray {

			if strings.Contains(k, j) {
				contains = true
				break
			}
		}
		if !contains {
			if strings.HasPrefix(k, customLabel.Name) {
				// Delete labels with prefix
				delete(namespace.Labels, k)
			}
		}
	}
	return nil
}
