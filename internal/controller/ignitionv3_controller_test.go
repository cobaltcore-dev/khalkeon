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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("IgnitionV3 Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			name               = "test-ignition"
			validConfigVersion = "3.5.0"
		)

		var (
			ign             *metalv1alpha1.IgnitionV3
			nn              = types.NamespacedName{Name: name, Namespace: namespace}
			deleteIfPresent = func(obj client.Object, opts ...func(client.Object)) {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
					return
				}
				for _, opt := range opts {
					opt(obj)
				}
				Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
			}

			withFinalizers = func(ign client.Object) {
				if controllerutil.RemoveFinalizer(ign, finalizer) {
					Expect(k8sClient.Update(ctx, ign)).To(Succeed())
				}
			}
		)

		BeforeEach(func() {
			ign = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		})

		AfterEach(func() {
			deleteIfPresent(ign, withFinalizers)
		})

		When("Ignition doesn't have target secret", func() {
			It("when configuration is valid, should update status", func() {
				ign.Spec.Config.Ignition.Version = validConfigVersion
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
				Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConfigurationType)).To(BeTrue())
			})

			It("when configuration is invalid, should update status", func() {
				ign.Spec.Config.Ignition.Version = "invalid"
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
				Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConfigurationType)).To(BeFalse())
			})
		})

		When("Ignition has target secret", func() {
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
				ign.Spec.Config.Ignition.Version = validConfigVersion
				ign.Spec.KernelArguments.ShouldExist = []metalv1alpha1.KernelArgument{"ignition-1 value"}
				ign.Spec.TargetSecret = &corev1.LocalObjectReference{Name: secretName}
				labels := map[string]string{"merge": "true"}
				ign.Spec.Config.Ignition.Config.Merge = &metav1.LabelSelector{MatchLabels: labels} // link to ign2

				secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace}}

				ign2 = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name2, Namespace: namespace}}
				ign2.Spec.Config.Ignition.Version = validConfigVersion
				ign2.Spec.KernelArguments.ShouldNotExist = []metalv1alpha1.KernelArgument{"ignition-2 value"}
				ign2.Labels = labels
				recLabels := map[string]string{"merge": "recurrently"}
				ign2.Spec.Config.Ignition.Config.Merge = &metav1.LabelSelector{MatchLabels: recLabels} // link to ign3

				ign3 = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name3, Namespace: namespace}}
				ign3.Spec.Config.Ignition.Version = validConfigVersion
				ign3.Spec.Passwd.Groups = []metalv1alpha1.PasswdGroup{{Name: "ignition-3 value"}}
				ign3.Labels = recLabels
			})

			AfterEach(func() {
				deleteIfPresent(secret)
				deleteIfPresent(ign2, withFinalizers)
				deleteIfPresent(ign3, withFinalizers)
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

			When("Ignition has replace field", func() {
				const (
					replaceName = "test-ignition-replace"
				)

				var (
					replaceIgn *metalv1alpha1.IgnitionV3
				)

				BeforeEach(func() {
					replaceIgn = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: replaceName, Namespace: namespace}}
					replaceIgn.Spec.Config.Ignition.Version = validConfigVersion
					replaceIgn.Spec.KernelArguments.ShouldExist = []metalv1alpha1.KernelArgument{"replace ignition value"}
					replaceIgn.Spec.Passwd.Groups = []metalv1alpha1.PasswdGroup{{Name: "replace ignition value"}}
				})

				AfterEach(func() {
					deleteIfPresent(replaceIgn, withFinalizers)
				})

				It("when an IgnitionV3 has a replace loop, should update the IgnitionV3 status to false", func() {
					replaceIgn.Spec.Config.Ignition.Config.Replace = &corev1.LocalObjectReference{Name: replaceName}
					Expect(k8sClient.Create(ctx, replaceIgn)).To(Succeed())

					controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
					_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
					Expect(err).NotTo(HaveOccurred())

					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(replaceIgn), replaceIgn)).To(Succeed())
					Expect(meta.IsStatusConditionTrue(replaceIgn.Status.Conditions, metalv1alpha1.ConfigurationType)).To(BeFalse())
				})

				It("when an IgnitionV3 with target secret is replaced with non existing IgnitionV3, should return an error", func() {
					ign.Spec.Config.Ignition.Config.Replace = &corev1.LocalObjectReference{Name: replaceName}
					ign.Spec.Config.Ignition.Config.Merge = nil
					Expect(k8sClient.Create(ctx, ign)).To(Succeed())

					controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
					_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
					Expect(err.Error()).To(Equal(`couldn't create merged configuration: couldn't get ignition. Reason: ignitionv3s.metal.cobaltcore.dev "test-ignition-replace" not found`))
				})

				It("when an IgnitionV3 with target secret is replaced with existing IgnitionV3, should create a secret with replaced config", func() {
					ign.Spec.Config.Ignition.Config.Replace = &corev1.LocalObjectReference{Name: replaceName}
					ign.Spec.Config.Ignition.Config.Merge = nil
					Expect(k8sClient.Create(ctx, ign)).To(Succeed())
					Expect(k8sClient.Create(ctx, replaceIgn)).To(Succeed())

					controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
					_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
					Expect(err).NotTo(HaveOccurred())

					Expect(k8sClient.Get(ctx, secretNn, secret)).To(Succeed())
					Expect(secret.Data[secretConfigData]).To(Equal([]byte(`{"ignition":{"config":{"replace":{"verification":{}}},"proxy":{},"security":{"tls":{}},"timeouts":{},"version":"3.5.0"},"kernelArguments":{"shouldExist":["replace ignition value"]},"passwd":{"groups":[{"name":"replace ignition value"}]},"storage":{},"systemd":{}}`)))
				})

				It("when an IgnitionV3 collected with merge is replaced with existing IgnitionV3, should create a secret with replaced config", func() {
					ign3.Spec.Config.Ignition.Config.Replace = &corev1.LocalObjectReference{Name: replaceName}
					Expect(k8sClient.Create(ctx, ign)).To(Succeed())
					Expect(k8sClient.Create(ctx, ign2)).To(Succeed())
					Expect(k8sClient.Create(ctx, ign3)).To(Succeed())
					Expect(k8sClient.Create(ctx, replaceIgn)).To(Succeed())

					controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
					_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ign3)})
					Expect(err).NotTo(HaveOccurred())
					_, err = controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
					Expect(err).NotTo(HaveOccurred())

					Expect(k8sClient.Get(ctx, secretNn, secret)).To(Succeed())
					Expect(secret.Data[secretConfigData]).To(Equal([]byte(`{"ignition":{"config":{"replace":{"verification":{}}},"proxy":{},"security":{"tls":{}},"timeouts":{},"version":"3.5.0"},"kernelArguments":{"shouldExist":["ignition-1 value","replace ignition value"],"shouldNotExist":["ignition-2 value"]},"passwd":{"groups":[{"name":"replace ignition value"}]},"storage":{},"systemd":{}}`)))
				})
			})
		})
	})
})
