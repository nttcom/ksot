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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	source "github.com/fluxcd/source-controller/api/v1beta2"
	deviceoperator "github.com/nttcom/kuesta/device-operator/api/v1alpha1"
	"github.com/nttcom/kuesta/pkg/testing/gnmihelper"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	provisioner "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceOperator controller", func() {
	ctx := context.Background()

	config1 := []byte(`{
	Interface: {
		Ethernet1: {
			Name:        "Ethernet1"
			Description: "foo"
		}
	}
}`)
	config2 := []byte(`{
		Interface: {
			Ethernet1: {
				Name:        "Ethernet1"
				Description: "bar"
			}
		}
	}`)
	rev1st := "rev1"
	rev2nd := "rev2"

	var testOpe deviceoperator.OcDemo
	Expect(testhelper.NewTestDataFromFixture("device1.deviceoperator", &testOpe)).NotTo(HaveOccurred())
	var testDr provisioner.DeviceRollout
	Expect(testhelper.NewTestDataFromFixture("devicerollout", &testDr)).NotTo(HaveOccurred())
	var testGr source.GitRepository
	Expect(testhelper.NewTestDataFromFixture("gitrepository", &testGr)).NotTo(HaveOccurred())

	BeforeEach(func() {
		Expect(k8sClient.Create(ctx, testOpe.DeepCopy())).NotTo(HaveOccurred())
		Expect(k8sClient.Create(ctx, testDr.DeepCopy())).NotTo(HaveOccurred())
		Expect(k8sClient.Create(ctx, testGr.DeepCopy())).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &deviceoperator.OcDemo{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &provisioner.DeviceRollout{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &source.GitRepository{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
	})

	It("should create subscriber pod", func() {
		var pod corev1.Pod
		Eventually(func() error {
			key := types.NamespacedName{
				Name:      fmt.Sprintf("subscriber-%s", testOpe.Name),
				Namespace: testOpe.Namespace,
			}
			if err := k8sClient.Get(ctx, key, &pod); err != nil {
				return err
			}
			return nil
		}, timeout, interval).Should(Succeed())
	})

	startRollout := func(config []byte, rev string) func() error {
		return func() error {
			var dr provisioner.DeviceRollout
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
				return err
			}
			dr.Status.Phase = provisioner.RolloutPhaseHealthy
			dr.Status.Status = provisioner.RolloutStatusRunning
			dr.Status.SetDeviceStatus(testOpe.Name, provisioner.DeviceStatusRunning)
			if dr.Status.DesiredDeviceConfigMap == nil {
				dr.Status.DesiredDeviceConfigMap = map[string]provisioner.DeviceConfig{}
			}
			dr.Status.DesiredDeviceConfigMap[testOpe.Name] = provisioner.DeviceConfig{
				Checksum:    testhelper.Hash(config),
				GitRevision: rev,
			}
			fmt.Fprintf(GinkgoWriter, "device rollout status, %+v\n", dr.Status)
			if err := k8sClient.Status().Update(ctx, &dr); err != nil {
				return err
			}
			return nil
		}
	}

	Context("when initializing without baseRevision", func() {
		BeforeEach(func() {
			checksum, buf := newGitRepoArtifact(func(dir string) {
				Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1)).NotTo(HaveOccurred())
			})
			data, _ := io.ReadAll(buf)
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write(data)
				Expect(err).NotTo(HaveOccurred())
			}))

			gr := testGr.DeepCopy()
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
			}, timeout, interval).Should(Succeed())
			gr.Status.Artifact = &source.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: rev1st,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, gr)
			}, timeout, interval).Should(Succeed())
		})

		It("should create subscriber pod", func() {
			Eventually(func() error {
				var pod corev1.Pod
				key := types.NamespacedName{
					Name:      fmt.Sprintf("subscriber-%s", testOpe.Name),
					Namespace: testOpe.Namespace,
				}
				if err := k8sClient.Get(ctx, key, &pod); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should initialize device resource with no base revision", func() {
			var ope deviceoperator.OcDemo
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope)).NotTo(HaveOccurred())
			Expect(ope.Status.BaseRevision).To(Equal(""))
			Expect(ope.Status.LastApplied).To(BeNil())
			Expect(ope.Status.Checksum).To(Equal(""))
		})

		Context("when device config updated", func() {
			It("should send gNMI SetRequest and change to completed when request succeeded", func() {
				setCalled := false
				m := &gnmihelper.GnmiMock{
					SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
						setCalled = true
						return &pb.SetResponse{}, nil
					},
				}
				lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testOpe.Spec.Address, testOpe.Spec.Port))
				Expect(err).NotTo(HaveOccurred())
				gs := gnmihelper.NewGnmiServerWithListener(m, lis)
				defer gs.Stop()

				Eventually(startRollout(config1, rev1st), timeout, interval).Should(Succeed())

				var dr provisioner.DeviceRollout
				Eventually(func() error {
					if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
						return err
					}
					if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
						return fmt.Errorf("status not changed yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())

				Expect(setCalled).To(BeTrue())
				Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusCompleted))
			})
		})
	})

	Context("when initializing with baseRevision", func() {
		BeforeEach(func() {
			checksum, buf := newGitRepoArtifact(func(dir string) {
				Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1)).NotTo(HaveOccurred())
			})
			data, _ := io.ReadAll(buf)
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write(data)
				Expect(err).NotTo(HaveOccurred())
			}))

			gr := testGr.DeepCopy()
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
			}, timeout, interval).Should(Succeed())
			gr.Status.Artifact = &source.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: rev1st,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, gr)
			}, timeout, interval).Should(Succeed())

			// set base revision
			var ope deviceoperator.OcDemo
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope); err != nil {
					return err
				}
				ope.Spec.BaseRevision = rev1st
				if err := k8sClient.Update(ctx, &ope); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope); err != nil {
					return err
				}
				if ope.Status.BaseRevision != rev1st {
					return fmt.Errorf("revision not updated yet: rev=%s", ope.Status.BaseRevision)
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should create subscriber pod", func() {
			Eventually(func() error {
				var pod corev1.Pod
				key := types.NamespacedName{
					Name:      fmt.Sprintf("subscriber-%s", testOpe.Name),
					Namespace: testOpe.Namespace,
				}
				if err := k8sClient.Get(ctx, key, &pod); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should initialize device resource with the specified base revision", func() {
			var ope deviceoperator.OcDemo
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope)).NotTo(HaveOccurred())
			Expect(ope.Status.BaseRevision).To(Equal(rev1st))
			Expect(ope.Status.LastApplied).To(Equal(config1))
			Expect(ope.Status.Checksum).To(Equal(testhelper.Hash(config1)))
		})

		It("should change rollout status to Completed when checksum is the same", func() {
			Eventually(startRollout(config1, rev2nd), timeout, interval).Should(Succeed())

			var dr provisioner.DeviceRollout
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
					return err
				}
				if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
					return fmt.Errorf("status not changed yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusCompleted))
		})

		It("should change rollout status to ChecksumError when checksum is mismatched", func() {
			checksum, buf := newGitRepoArtifact(func(dir string) {
				Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), []byte("mismatched"))).NotTo(HaveOccurred())
			})
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, buf)
				Expect(err).NotTo(HaveOccurred())
			}))

			gr := testGr.DeepCopy()
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
			}, timeout, interval).Should(Succeed())
			gr.Status.Artifact = &source.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: rev2nd,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, gr)
			}, timeout, interval).Should(Succeed())

			Eventually(startRollout(config2, rev2nd), timeout, interval).Should(Succeed())

			var dr provisioner.DeviceRollout
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
					return err
				}
				if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
					return fmt.Errorf("status not changed yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusChecksumError))
		})

		Context("when device config updated", func() {
			BeforeEach(func() {
				checksum, buf := newGitRepoArtifact(func(dir string) {
					Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config2)).NotTo(HaveOccurred())
				})
				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, err := io.Copy(w, buf)
					Expect(err).NotTo(HaveOccurred())
				}))

				gr := testGr.DeepCopy()
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)).NotTo(HaveOccurred())
				gr.Status.Artifact = &source.Artifact{
					URL:      h.URL,
					Checksum: checksum,
					Revision: rev2nd,
				}
				Eventually(func() error {
					return k8sClient.Status().Update(ctx, gr)
				}, timeout, interval).Should(Succeed())
			})

			It("should send gNMI SetRequest and change to completed when request succeeded", func() {
				setCalled := false
				m := &gnmihelper.GnmiMock{
					SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
						setCalled = true
						return &pb.SetResponse{}, nil
					},
				}
				lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testOpe.Spec.Address, testOpe.Spec.Port))
				Expect(err).NotTo(HaveOccurred())
				gs := gnmihelper.NewGnmiServerWithListener(m, lis)
				defer gs.Stop()

				Eventually(startRollout(config2, rev2nd), timeout, interval).Should(Succeed())

				var dr provisioner.DeviceRollout
				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
						return fmt.Errorf("status not changed yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())

				Expect(setCalled).To(BeTrue())
				Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusCompleted))
			})

			It("should send gNMI SetRequest and change to failed when request failed", func() {
				setCalled := false
				m := &gnmihelper.GnmiMock{
					SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
						setCalled = true
						return &pb.SetResponse{}, fmt.Errorf("failed")
					},
				}
				lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testOpe.Spec.Address, testOpe.Spec.Port))
				Expect(err).NotTo(HaveOccurred())
				gs := gnmihelper.NewGnmiServerWithListener(m, lis)
				defer gs.Stop()

				Eventually(startRollout(config2, rev2nd), timeout, interval).Should(Succeed())

				var dr provisioner.DeviceRollout
				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
						return fmt.Errorf("status not changed yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())

				Expect(setCalled).To(BeTrue())
				Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusFailed))
			})
		})
	})
})

func newGitRepoArtifact(fn func(dir string)) (string, io.Reader) {
	dir, err := os.MkdirTemp("", "git-watcher-test-*")
	defer os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
	fn(dir)
	return testhelper.MustGenTgzArchiveDir(dir)
}
