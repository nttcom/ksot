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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/nttcom/kuesta/pkg/testing/testhelper"
	kuestav1alpha1 "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GitRepository watcher", func() {
	ctx := context.Background()

	var testGr sourcev1.GitRepository
	Expect(testhelper.NewTestDataFromFixture("gitrepository", &testGr)).NotTo(HaveOccurred())

	config1 := []byte("foo")
	config2 := []byte("bar")
	revision := "test-rev"

	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "git-watcher-test-*")
		Expect(err).NotTo(HaveOccurred())
		Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1)).NotTo(HaveOccurred())
		Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2)).NotTo(HaveOccurred())

		gr := testGr.DeepCopy()
		Expect(k8sClient.Create(ctx, gr)).NotTo(HaveOccurred())

		checksum, buf := testhelper.MustGenTgzArchiveDir(dir)
		h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(w, buf); err != nil {
				panic(err)
			}
		}))

		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
		}, timeout, interval).Should(Succeed())
		gr.Status.Artifact = &sourcev1.Artifact{
			URL:      h.URL,
			Checksum: checksum,
			Revision: revision,
		}
		Eventually(func() error {
			return k8sClient.Status().Update(ctx, gr)
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &kuestav1alpha1.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &sourcev1.GitRepository{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
		os.RemoveAll(dir)
	})

	It("should create DeviceRollout", func() {
		var dr kuestav1alpha1.DeviceRollout
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr)
		}, timeout, interval).Should(Succeed())

		want := kuestav1alpha1.DeviceConfigMap{
			"device1": kuestav1alpha1.DeviceConfig{
				Checksum:    testhelper.Hash(config1),
				GitRevision: revision,
			},
			"device2": kuestav1alpha1.DeviceConfig{
				Checksum:    testhelper.Hash(config2),
				GitRevision: revision,
			},
		}
		Expect(dr.Spec.DeviceConfigMap).To(Equal(want))
	})

	Context("when device config updated", func() {
		config1 := []byte("foo-updated")
		config2 := []byte("bar-updated")
		revision := "test-rev-updated"

		BeforeEach(func() {
			var dr kuestav1alpha1.DeviceRollout
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr)
			}, timeout, interval).Should(Succeed())

			Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1)).NotTo(HaveOccurred())
			Expect(testhelper.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2)).NotTo(HaveOccurred())

			checksum, buf := testhelper.MustGenTgzArchiveDir(dir)
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(w, buf); err != nil {
					panic(err)
				}
			}))

			var gr sourcev1.GitRepository
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), &gr)).NotTo(HaveOccurred())
			gr.Status.Artifact = &sourcev1.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: revision,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, &gr)
			}, timeout, interval).Should(Succeed())
		})

		It("should update DeviceRollout", func() {
			var dr kuestav1alpha1.DeviceRollout
			Eventually(func() error {
				k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr)
				if dr.Spec.DeviceConfigMap["device1"].GitRevision != revision {
					return fmt.Errorf("not updated yet: %+v\n", dr.Spec.DeviceConfigMap)
				}
				return nil
			}, timeout, interval).Should(Succeed())

			want := kuestav1alpha1.DeviceConfigMap{
				"device1": kuestav1alpha1.DeviceConfig{
					Checksum:    testhelper.Hash(config1),
					GitRevision: revision,
				},
				"device2": kuestav1alpha1.DeviceConfig{
					Checksum:    testhelper.Hash(config2),
					GitRevision: revision,
				},
			}
			Expect(dr.Spec.DeviceConfigMap).To(Equal(want))
		})
	})
})
