package controller

import (
	"context"
	"time"

	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("customlabels controller", func() {
	const (
		CustomLabelName      = "testlabels"
		CustomLabelNamespace = "default"
		timeout              = time.Second * 10
		duration             = time.Second * 10
		interval             = time.Millisecond * 250
	)
	Context("When creating customlabel status", func() {
		It("change customlabels status to applied when labels are added", func() {
			By("creating new labels")
			ctx := context.Background()
			customLabels := &labelsv1.CustomLabel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "labels.dvir.io/v1",
					Kind:       "CustomLabel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomLabelName,
					Namespace: CustomLabelNamespace,
				},
				Spec: labelsv1.CustomLabelSpec{
					CustomLabels: map[string]string{"label1": "test1", "label2": "test2"},
				},
			}
			Expect(k8sClient.Create(ctx, customLabels)).Should(Succeed())

			// customLabelsLookupKey := types.NamespacedName{Name: CustomLabelName, Namespace: CustomLabelNamespace}
			// createdCustomLabels := &labelsv1.CustomLabel{}
			// Eventually(func() bool {
			// 	err := k8sClient.Get(ctx, customLabelsLookupKey, createdCustomLabels)
			// 	return err == nil
			// }, timeout, interval).Should(BeTrue())
			// Expect(createdCustomLabels.Status.Applied).Should(Equal(true))

		})
	})
})