package controller

import (
	"context"
	"fmt"
	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

// HandleDelete deletes labels from namespace
func (r *CustomLabelReconciler) HandleDelete(ctx context.Context, customLabels *labelsv1.CustomLabel, namespace *corev1.Namespace) (ctrl.Result, error) {
	//check if deleting protected labels and delete labels
	r.DeleteNameSpaceLabels(customLabels, namespace)
	// remove labels from namespace
	if err := r.Client.Update(ctx, namespace); err != nil {
		if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, err.Error(), map[string]labelsv1.LabelStatus{}); err != nil {
			return ctrl.Result{}, statusErr
		}

		return ctrl.Result{}, err

	}
	r.Log.Info("deleted labels from namespace")
	_, err := r.DeleteFinalizer(ctx, customLabels, r.Log)
	if err != nil {
		if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, err.Error(), map[string]labelsv1.LabelStatus{}); err != nil {
			return ctrl.Result{}, statusErr
		}

		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

// AddFinalizer Checks if the given CustomLabels CRD has the DeleteLabelsFinalizer
// Returns true if finalizer did not exist and was added
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
	}

	return false, nil

}

// DeleteFinalizer Checks if the given CRD contains the DeleteLabelsFinalizer and removes it.
// Returns true if finalizer existed and was removed
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
	}
	//Finalizer already deleted
	return false, nil
}

// AddNamespaceLabels Adds the labels in the spec of the given NamespaceLabel CRD to the given namespace
func (r *CustomLabelReconciler) AddNamespaceLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace, protectedPrefixArray []string, labelsToAdd map[string]string) map[string]labelsv1.LabelStatus {
	labelStatusMap := map[string]labelsv1.LabelStatus{}

	for k, v := range customLabel.Spec.CustomLabels {
		_, ok := labelsToAdd[k]
		labelStatus := &labelsv1.LabelStatus{}
		if !ok {
			r.Log.Info(fmt.Sprintf("not adding label: %s", k))
			labelStatus.Applied = false
			labelStatus.Value = v
			labelStatusMap[k] = *labelStatus
			continue
		}
		if _, nok := namespace.Labels[k]; nok {
			r.Log.Info(fmt.Sprintf("label already exists: %s", k))
			labelStatus.Applied = false
			labelStatus.Value = v
			labelStatusMap[k] = *labelStatus
			continue
		}
		var valid = true

		// Skip protected labels that contain a protected prefix
		for _, j := range protectedPrefixArray {
			if strings.Contains(k, j) {
				r.Log.Info(fmt.Sprintf("attemting to add a label with a protected prefix: %s", j))
				valid = false
				labelStatus.Applied = false
				labelStatus.Value = v
				labelStatusMap[k] = *labelStatus
				break

			}
		}
		if !valid {
			continue
		}
		// Add label to namespace
		namespace.Labels[k] = v
		labelStatus.Applied = true
		labelStatus.Value = v
		r.Log.Info(fmt.Sprintf("added label to namespace: %s", k))
		labelStatusMap[k] = *labelStatus

	}
	return labelStatusMap
}

// UpdateNamespace edits namespace with new labels and returns any errors
func (r *CustomLabelReconciler) UpdateNamespace(ctx context.Context, customLabels *labelsv1.CustomLabel, namespace *corev1.Namespace) error {
	if err := r.Client.Update(ctx, namespace); err != nil {
		r.Log.Error("error adding labels", zap.Error(err))
		customLabels.Status.Message = "error adding labels to namespace"
		if err := r.Client.Status().Update(ctx, customLabels); err != nil {
			r.Log.Error("unable to modify custom label status", zap.Error(err))
			if statusErr := r.UpdateCustomLabelStatus(ctx, customLabels, err.Error(), map[string]labelsv1.LabelStatus{}); err != nil {
				return statusErr
			}

			return err
		}
		return err
	}
	return nil

}

// ParseLabels go through PerLabelStatus of crd to check if labels have already been applied.
// Change labels accordingly
func (r *CustomLabelReconciler) ParseLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace) map[string]string {
	lastLabelState := customLabel.Status.PerLabelStatus
	if len(lastLabelState) == 0 {
		r.Log.Info("no label status, CRD is new")
		return customLabel.Spec.CustomLabels
	}
	labelsToAdd := map[string]string{}
	for k, v := range customLabel.Spec.CustomLabels {
		j, ok := lastLabelState[k]
		if !ok {
			// label controlled by another CRD
			if _, lok := namespace.Labels[k]; lok {
				r.Log.Info(fmt.Sprintf("Label already exists: %s", k))
				continue
			} else {
				//new label
				labelsToAdd[k] = v
				continue
			}
		}
		if j.Applied {
			if j.Value != v {
				// Label with edited value
				r.Log.Info(fmt.Sprintf("Applied label was changed: %s", k))
				labelsToAdd[k] = v
				continue
			}
			//edited value
			r.Log.Info(fmt.Sprintf("Applied label unchanged, skipping: %s", k))
			continue

		}

	}

	return labelsToAdd
}

// DeleteLabels delete from namespace applied labels that are not in spec anymore
func (r *CustomLabelReconciler) DeleteLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace) {
	for a, b := range customLabel.Status.PerLabelStatus {
		if b.Applied {
			// label was deleted from crd
			_, ok := customLabel.Labels[a]
			if !ok {
				r.Log.Info(fmt.Sprintf("Applied label was deleted: %s", a))
				delete(namespace.Labels, a)

			}
		}
	}
}

// DeleteNameSpaceLabels Deletes the given namespace labels from the given namespace
// Will only delete labels that exist in the namespace with the same value as in the label CRD
func (r *CustomLabelReconciler) DeleteNameSpaceLabels(customLabel *labelsv1.CustomLabel, namespace *corev1.Namespace) {
	for k, v := range namespace.ObjectMeta.Labels {
		j, ok := customLabel.Spec.CustomLabels[k]
		if ok && v == j && customLabel.Status.PerLabelStatus[k].Applied {
			// Delete labels with that exist in the CRD and that have the same value
			delete(namespace.Labels, k)
		}
	}
}

// UpdateCustomLabelStatus Updates the status of the CRD with any errors that occured or if it succeeded
func (r *CustomLabelReconciler) UpdateCustomLabelStatus(ctx context.Context, CustomLabel *labelsv1.CustomLabel, message string, labelStatus map[string]labelsv1.LabelStatus) error {
	CustomLabel.Status.Message = message
	CustomLabel.Status.PerLabelStatus = labelStatus
	if err := r.Client.Status().Update(ctx, CustomLabel); err != nil {
		r.Log.Error(fmt.Sprintf("unable to modify custom label status: %s", CustomLabel.Name), zap.Error(err))
		return err
	}
	return nil
}