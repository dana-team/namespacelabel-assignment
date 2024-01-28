package controller

import (
	"context"
	labelsv1 "github.com/dvirgilad/namespacelabel-assignment/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
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
				for k := range labelsToAdd {
					_, ok := searchNameSpace.Labels[k]
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
		It("Labels are deleted from namespace when CRD is deleted", func() {
			By("deleting labels")
			customLabelsLookupKey := types.NamespacedName{Name: CustomLabelName, Namespace: CustomLabelNamespace}
			createdCustomLabels := &labelsv1.CustomLabel{}
			Expect(k8sClient.Get(ctx, customLabelsLookupKey, createdCustomLabels)).Should(BeNil(), "should find resource")

			deleteOpts := &client.DeleteOptions{}
			err := k8sClient.Delete(ctx, createdCustomLabels, deleteOpts)
			Expect(err).To(BeNil())
			namespaceLookupKey := types.NamespacedName{Name: CustomLabelNamespace}
			searchNameSpace := &corev1.Namespace{}
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespaceLookupKey, searchNameSpace)).Should(BeNil(), "should find namespace")
				for k := range createdCustomLabels.Spec.CustomLabels {
					_, ok := searchNameSpace.Labels[k]
					if ok {
						return false
					} else {
						continue
					}
				}
				return true
			}, timeout, interval,
			).Should(BeTrue(), "all labels in CR should be deleted from namespace")
		})
		It("adding two label CRDs", func() {
			By("adding labels")
			labelsToAdd := map[string]string{"label1": "test1", "label2": "test2"}
			moreLabelsToAdd := map[string]string{"label2": "test3"}
			customLabels := &labelsv1.CustomLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomLabelName,
					Namespace: CustomLabelNamespace,
				},
				Spec: labelsv1.CustomLabelSpec{
					CustomLabels: labelsToAdd,
				},
			}
			moreCustomLabels := &labelsv1.CustomLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomLabelName + "1",
					Namespace: CustomLabelNamespace,
				},
				Spec: labelsv1.CustomLabelSpec{
					CustomLabels: moreLabelsToAdd,
				},
			}
			err := k8sClient.Create(ctx, customLabels)
			Expect(err).ToNot(HaveOccurred())
			err = k8sClient.Create(ctx, moreCustomLabels)
			Expect(err).ToNot(HaveOccurred())
			customLabelsLookupKey := types.NamespacedName{Name: CustomLabelName, Namespace: CustomLabelNamespace}
			createdCustomLabels := &labelsv1.CustomLabel{}
			moreCustomLabelsLookupKey := types.NamespacedName{Name: CustomLabelName + "1", Namespace: CustomLabelNamespace}
			moreCreatedCustomLabels := &labelsv1.CustomLabel{}
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, customLabelsLookupKey, createdCustomLabels)).Should(BeNil(), "should find resource")
				return createdCustomLabels.Status.Applied
			},
				timeout, interval,
			).Should(BeTrue(), "CR status.Applied of first crd should be true")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, moreCustomLabelsLookupKey, moreCreatedCustomLabels)).Should(BeNil(), "should find resource")
				return moreCreatedCustomLabels.Status.Applied
			},
				timeout, interval,
			).Should(BeTrue(), "CR status.Applied of second crd should be True")
			namespaceLookupKey := types.NamespacedName{Name: CustomLabelNamespace}
			searchNameSpace := &corev1.Namespace{}
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, namespaceLookupKey, searchNameSpace)).Should(BeNil(), "should find namespace")
				for k := range labelsToAdd {
					v, ok := searchNameSpace.Labels[k]
					if ok && labelsToAdd[k] == v {
						continue
					} else {
						return false
					}
				}
				return true
			}, timeout, interval,
			).Should(BeTrue(), "all labels in namespace should be from first CRD")
		})

	})

})
