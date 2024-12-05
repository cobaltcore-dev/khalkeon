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
	"fmt"

	metalv1alpha1 "github.com/cobaltcore-dev/khalkeon/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("IgnitionV3 Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			name       = "test-ignition"
			secretName = "test-ignition-secret"
		)

		var (
			ign *metalv1alpha1.IgnitionV3
			nn  types.NamespacedName
		)

		BeforeEach(func() {
			ign = &metalv1alpha1.IgnitionV3{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
			nn = client.ObjectKeyFromObject(ign)
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, ign)).To(Succeed())
		})

		// When("Igntion doesn't have target secret", func() {
		// 	It("when configuration is valid, should update status ", func() {
		// 		ign.Spec.Config.Ignition.Version = "3.5.0"
		// 		Expect(k8sClient.Create(ctx, ign)).To(Succeed())

		// 		controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		// 		_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
		// 		Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConditionType)).To(BeTrue())
		// 	})

		// 	It("when configuration is invalid, should update status ", func() {
		// 		ign.Spec.Config.Ignition.Version = "invalid"
		// 		Expect(k8sClient.Create(ctx, ign)).To(Succeed())

		// 		controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		// 		_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		// 		Expect(err).NotTo(HaveOccurred())

		// 		Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
		// 		Expect(meta.IsStatusConditionTrue(ign.Status.Conditions, metalv1alpha1.ConditionType)).To(BeFalse())
		// 	})
		// })

		When("Igntion has target secret", func() {
			BeforeEach(func() {
				ign.Spec.Config.Ignition.Version = "3.5.0"
				ign.Spec.TargetSecret = &v1.LocalObjectReference{Name: secretName}
			})

			It("when merge is empty, should create a secret with single config", func() {
				fmt.Println("test create", nn, "\""+ign.Namespace+"\"") //test create test-namespace/test-ignition "test-namespace"
				Expect(k8sClient.Create(ctx, ign)).To(Succeed())

				controller := &IgnitionV3Reconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
				_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, nn, ign)).To(Succeed())
			})
		})
	})
})
