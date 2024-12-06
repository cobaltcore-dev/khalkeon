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
	metalv1alpha1 "github.com/cobaltcore-dev/khalkeon/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("IgnitionV3 Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			name = "test-ignition"
		)

		var (
			ign *metalv1alpha1.IgnitionV3
			nn  = types.NamespacedName{Name: name, Namespace: namespace}
		)

		BeforeEach(func() {
			ign = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, ign)).To(Succeed())
		})

		When("Igntion doesn't have target secret", func() {
			It("when configuration is valid, should update status ", func() {
				ign.Spec.Config.Ignition.Version = "3.5.0"
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
				Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConditionType)).To(BeTrue())
			})

			It("when configuration is invalid, should update status ", func() {
				ign.Spec.Config.Ignition.Version = "invalid"
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
				Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConditionType)).To(BeFalse())
			})
		})

		When("Igntion has target secret", func() {
			const (
				secretName = "test-ignition-secret"
				name2      = "test-ignition-2"
				name3      = "test-ignition-3"
			)

			var (
				secret   *corev1.Secret
				secretNn = types.NamespacedName{Name: secretName, Namespace: namespace}
				ign2     *metalv1alpha1.IgnitionV3
				ign3     *metalv1alpha1.IgnitionV3
			)

			BeforeEach(func() {
				ign.Spec.Config.Ignition.Version = "3.5.0"
				ign.Spec.KernelArguments.ShouldExist = []metalv1alpha1.KernelArgument{"ignition-1 value"}
				ign.Spec.TargetSecret = &corev1.LocalObjectReference{Name: secretName}
				labels := map[string]string{"merge": "true"}
				ign.Spec.Config.Ignition.Config = new(metalv1alpha1.IgnitionConfig)
				ign.Spec.Config.Ignition.Config.Merge = metav1.LabelSelector{MatchLabels: labels} // link to ign2

				secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace}}

				ign2 = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name2, Namespace: namespace}}
				ign2.Spec.Config.Ignition.Version = "3.5.0"
				ign2.Spec.KernelArguments.ShouldNotExist = []metalv1alpha1.KernelArgument{"ignition-2 value"}
				ign2.Labels = labels
				recLabels := map[string]string{"merge": "recurrently"}
				ign2.Spec.Config.Ignition.Config = new(metalv1alpha1.IgnitionConfig)
				ign2.Spec.Config.Ignition.Config.Merge = metav1.LabelSelector{MatchLabels: recLabels} // link to ign3

				ign3 = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name3, Namespace: namespace}}
				ign3.Spec.Config.Ignition.Version = "3.5.0"
				ign3.Spec.Passwd.Groups = []metalv1alpha1.PasswdGroup{{Name: "ignition-3 value"}}
				ign3.Labels = recLabels
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
				k8sClient.Delete(ctx, ign2)
				k8sClient.Delete(ctx, ign3)
			})

			It("when merge is empty, should create a secret with single config", func() {
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, secretNn, secret)).To(Succeed())
				Expect(secret.Data[secretConfigData]).To(Equal([]byte(`{"ignition":{"config":{"replace":{"verification":{}}},"proxy":{},"security":{"tls":{}},"timeouts":{},"version":"3.5.0"},"kernelArguments":{"shouldExist":["ignition-1 value"]},"passwd":{},"storage":{},"systemd":{}}`)))
			})

			It("when merge is not empty, should create a secret with merged config", func() {
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())
				Expect(k8sClient.Create(ctx, ign2)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, secretNn, secret)).To(Succeed())
				Expect(secret.Data[secretConfigData]).To(Equal([]byte(`{"ignition":{"config":{"replace":{"verification":{}}},"proxy":{},"security":{"tls":{}},"timeouts":{},"version":"3.5.0"},"kernelArguments":{"shouldExist":["ignition-1 value"],"shouldNotExist":["ignition-2 value"]},"passwd":{},"storage":{},"systemd":{}}`)))
			})

			It("when merge IgnitionV3 are collected recurrently, should create a secret with merged config", func() {
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())
				Expect(k8sClient.Create(ctx, ign2)).To(Succeed())
				Expect(k8sClient.Create(ctx, ign3)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, secretNn, secret)).To(Succeed())
				Expect(secret.Data[secretConfigData]).To(Equal([]byte(`{"ignition":{"config":{"replace":{"verification":{}}},"proxy":{},"security":{"tls":{}},"timeouts":{},"version":"3.5.0"},"kernelArguments":{"shouldExist":["ignition-1 value"],"shouldNotExist":["ignition-2 value"]},"passwd":{"groups":[{"name":"ignition-3 value"}]},"storage":{},"systemd":{}}`)))
			})
		})
	})
})
