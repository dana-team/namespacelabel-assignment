package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("customlabels controller", func() {
	const (
		CustomLabelName      = "testlabels"
		CustomLabelNamespace = "testnamespace"
		timeout              = time.Second * 10
		duration             = time.Second * 10
		interval             = time.Millisecond * 250
	)
	Context("When creating customlabel status", func() {
		It("change customlabels status to applied when labels are added", func() {
			By("creating new labels")
			ctx := context.Background()
			labelsToAdd := map[string]string{"label1": "test1", "label2": "test2"}
			customLabels := &labelsv1.CustomLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomLabelName,
					Namespace: CustomLabelNamespace,
				},
				Spec: labelsv1.CustomLabelSpec{
					CustomLabels: labelsToAdd,
				},
			}
			err := k8sClient.Create(ctx, customLabels)
			Expect(err).ToNot(HaveOccurred())

			customLabelsLookupKey := types.NamespacedName{Name: CustomLabelName, Namespace: CustomLabelNamespace}
			createdCustomLabels := &labelsv1.CustomLabel{}
			namespaceLookupKey := types.NamespacedName{Name: CustomLabelNamespace}
			searchNameSpace := &corev1.Namespace{}

			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, customLabelsLookupKey, createdCustomLabels)).Should(BeNil(), "should find resource")
				return createdCustomLabels.Status.Applied
			},
				timeout, interval,
			).Should(BeTrue(), "CR status.Applied should be true")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespaceLookupKey, searchNameSpace)).Should(BeNil(), "should find namespace")
				for k, _ := range labelsToAdd {
					_, ok := searchNameSpace.Labels[fmt.Sprintf("%s/%s", CustomLabelName, k)]
					if ok {
						continue
					} else {
						return false
					}
				}
				return true
			}, timeout, interval,
			).Should(BeTrue(), "all labels in CR should be added to namespace")

		})
	})
})
