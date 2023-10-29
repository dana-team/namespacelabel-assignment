package controller

import (
	"context"
	"time"

	labelsv1alpha1 "github.com/dvirgilad/namespacelabel-assignment/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("customlabels controller", func() {
	const (
		CustomLabelsName      = "testlabels"
		CustomLabelsNamespace = "default"
		timeout               = time.Second * 10
		duration              = time.Second * 10
		interval              = time.Millisecond * 250
	)
	Context("When creating customlabel status", func() {
		It("change customlabels status to applied when labels are added", func() {
			By("creating new labels")
			ctx := context.Background()
			customLabels := &labelsv1alpha1.CustomLabels{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "labels.my.domain/v1alpha1",
					Kind:       "CustomLabels",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomLabelsName,
					Namespace: CustomLabelsNamespace,
				},
				Spec: labelsv1alpha1.CustomLabelsSpec{
					CustomLabels: map[string]string{"label1": "test1", "label2": "test2"},
				},
			}
			Expect(k8sClient.Create(ctx, customLabels)).Should(Succeed())

			customLabelsLookupKey := types.NamespacedName{Name: CustomLabelsName, Namespace: CustomLabelsNamespace}
			createdCustomLabels := &labelsv1alpha1.CustomLabels{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, customLabelsLookupKey, createdCustomLabels)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdCustomLabels.Status.Applied).Should(Equal(true))

		})
	})
})
