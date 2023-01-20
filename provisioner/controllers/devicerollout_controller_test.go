/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package controllers_test

import (
	"context"
	"fmt"

	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	provisioner "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceRollout controller", func() {
	ctx := context.Background()

	var testDr provisioner.DeviceRollout
	Expect(testhelper.NewTestDataFromFixture("devicerollout", &testDr)).NotTo(HaveOccurred())
	desired := provisioner.DeviceConfigMap{
		"device1": {Checksum: "desired", GitRevision: "desired"},
		"device2": {Checksum: "desired", GitRevision: "desired"},
	}

	BeforeEach(func() {
		err := k8sClient.Create(ctx, testDr.DeepCopy())
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			var dr provisioner.DeviceRollout
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
				return err
			}
			if dr.Status.Phase == "" {
				return fmt.Errorf("not updated yet")
			}
			return nil
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &provisioner.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should update DeviceRollout's status to running", func() {
		var dr provisioner.DeviceRollout
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
		Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
		Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
		Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(dr.Spec.DeviceConfigMap))
		for _, v := range dr.Status.DeviceStatusMap {
			Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
		}
	})

	Context("when devices update succeeded", func() {
		BeforeEach(func() {
			var dr provisioner.DeviceRollout
			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				for k := range dr.Status.DeviceStatusMap {
					dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusCompleted
				}
				return k8sClient.Status().Update(ctx, &dr)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				if dr.Status.Status == provisioner.RolloutStatusRunning {
					return fmt.Errorf("not updated yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should update DeviceRollout's status to completed", func() {
			var dr provisioner.DeviceRollout
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
			Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
			Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusCompleted))
		})

		Context("when new config provisioned", func() {
			BeforeEach(func() {
				var dr provisioner.DeviceRollout
				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					dr.Spec.DeviceConfigMap = desired
					return k8sClient.Update(ctx, &dr)
				}, timeout, interval).Should(Succeed())

				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					if dr.Status.Status == provisioner.RolloutStatusCompleted {
						return fmt.Errorf("not updated yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())
			})

			It("should update DeviceRollout's status to running", func() {
				var dr provisioner.DeviceRollout
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
				Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
				Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
				Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
				for _, v := range dr.Status.DeviceStatusMap {
					Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
				}
			})

			Context("when device update failed", func() {
				BeforeEach(func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusFailed
							break
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())

					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Phase == provisioner.RolloutPhaseHealthy {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())
				})

				It("should update DeviceRollout's phase to rollback and status to running", func() {
					var dr provisioner.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
					Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
					Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
					for _, v := range dr.Status.DeviceStatusMap {
						Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
					}
				})

				It("should update DeviceRollout to rollback/completed when rollback succeeded", func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusCompleted
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == provisioner.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusCompleted))
				})

				It("should update DeviceRollout to rollback/failed when rollback failed", func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusFailed
							break
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == provisioner.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusFailed))
				})
			})
		})
	})
})
