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

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	fluxcd "github.com/fluxcd/source-controller/api/v1beta2"
	deviceoperator "github.com/nttcom/kuesta/device-operator/api/v1alpha1"
	"github.com/nttcom/kuesta/device-operator/internal"
	"github.com/nttcom/kuesta/device-operator/internal/model"
	"github.com/nttcom/kuesta/pkg/artifact"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	device "github.com/nttcom/kuesta/pkg/device"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/stacktrace"
	provisioner "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	gnmiproto "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/prototext"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	subscriberImage        string
	subscriberImageVersion string
	aggregatorUrl          string
)

func SetupEnv() {
	subscriberImage = internal.MustGetEnv("KUESTA_SUBSCRIBER_IMAGE")
	subscriberImageVersion = internal.MustGetEnv("KUESTA_SUBSCRIBER_IMAGE_VERSION")
	aggregatorUrl = internal.MustGetEnv("KUESTA_AGGREGATOR_URL")
}

func (r *DeviceReconciler) DoReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconciliation")

	if err := r.createSubscriberPodIfNotExist(ctx, req.NamespacedName); err != nil {
		r.Error(ctx, err, "create subscriberPod")
		return ctrl.Result{}, err
	}

	device, err := r.getDevice(ctx, req.NamespacedName)
	if err != nil {
		r.Error(ctx, err, "get Device resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// force set checksum and lastApplied config when baseRevision updated
	if device.Spec.BaseRevision != device.Status.BaseRevision {
		if err := r.forceReplaceLastApplied(ctx, req); err != nil {
			r.Error(ctx, err, "force update status to the one given by baseRevision")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	if device.Status.LastApplied == nil {
		if device.Spec.DiffOnly {
			l.Info("reconcile stopped: lastApplied config is not set. you must initialize lastApplied config to update device config automatically")
			return ctrl.Result{}, nil
		}
		l.Info("since lastApplied config is not set, all configs will be pushed without purging deleted fields. To stop this behaviour, set spec.DiffOnly to true.")
	}

	var dr provisioner.DeviceRollout
	if err := r.Get(ctx, types.NamespacedName{Namespace: device.Namespace, Name: device.Spec.RolloutRef}, &dr); err != nil {
		r.Error(ctx, err, "get DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if dr.Status.GetDeviceStatus(device.Name) != provisioner.DeviceStatusRunning {
		l.Info("reconcile skipped: device status is not running")
		return ctrl.Result{}, nil
	}

	next := dr.Status.ResolveNextDeviceConfig(device.Name)
	if next == nil || next.Checksum == "" {
		l.Info("device data is not stored at git repository")
		return ctrl.Result{}, nil
	}
	if next.Checksum == device.Status.Checksum {
		l.Info(fmt.Sprintf("already provisioned: revision=%s", next.GitRevision))
		if err := r.updateRolloutStatus(ctx, dr, device.Name, provisioner.DeviceStatusCompleted); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	l.Info(fmt.Sprintf("next: revision=%s", next.GitRevision))

	var gr fluxcd.GitRepository
	if err := r.Get(ctx, types.NamespacedName{Namespace: dr.Namespace, Name: dr.Name}, &gr); err != nil {
		r.Error(ctx, err, "get GitRepository")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	dp, checksum, err := fetchArtifact(ctx, gr, device.Name, "")
	if err != nil {
		r.Error(ctx, err, "failed to fetch device config. re-check after 10 seconds")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	defer os.RemoveAll(dp.RootDir)
	if checksum != next.Checksum {
		err = fmt.Errorf("checksum is different: want=%s, got=%s", next.Checksum, checksum)
		r.Error(ctx, err, "check checksum")
		if err := r.updateRolloutStatus(ctx, dr, device.Name, provisioner.DeviceStatusChecksumError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	newBuf, err := dp.ReadDeviceConfigFile()
	if err != nil {
		r.Error(ctx, err, "read device config")
		return ctrl.Result{}, err
	}
	sr, err := makeSetRequest(newBuf, device.Status.LastApplied)
	if err != nil {
		r.Error(ctx, err, "make gnmi SetRequest")
		return ctrl.Result{}, err
	}
	l.V(1).Info("gnmi SetRequest payload", "updated", sr.GetUpdate(), "deleted", sr.GetDelete())

	var secret core.Secret
	var tlsData, credData map[string][]byte

	if device.Spec.TLS.SecretName != "" {
		if err := r.Get(ctx, types.NamespacedName{Namespace: device.Namespace, Name: device.Spec.TLS.SecretName}, &secret); err != nil {
			r.Error(ctx, err, "get secret for TLS", "secretName", device.Spec.TLS.SecretName)
			return ctrl.Result{}, nil
		}
		tlsData = secret.Data
	}
	if device.Spec.ConnectionInfo.SecretName != "" {
		if err := r.Get(ctx, types.NamespacedName{Namespace: device.Namespace, Name: device.Spec.ConnectionInfo.SecretName}, &secret); err != nil {
			r.Error(ctx, err, "get secret for credential", "secretName", device.Spec.ConnectionInfo.SecretName)
			return ctrl.Result{}, nil
		}
		credData = secret.Data
	}
	dest, err := device.Spec.GnmiDestination(tlsData, credData)
	if err != nil {
		r.Error(ctx, err, "make gnmi SetRequest")
		return ctrl.Result{}, err
	}

	var c gclient.Impl
	for i := 0; i < 3; i++ {
		if c, err = gnmiclient.New(ctx, dest); err == nil {
			break
		}
	}
	if err != nil {
		r.Error(ctx, err, "failed to create gNMI client and connect for 3 times. mark as ConnectionError")
		if err := r.updateRolloutStatus(ctx, dr, device.Name, provisioner.DeviceStatusConnectionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	defer c.Close()

	resp, gnmiSetErr := c.(*gnmiclient.Client).Set(ctx, sr)

	if gnmiSetErr != nil {
		r.Error(ctx, gnmiSetErr, "apply Set")
		if err := r.updateRolloutStatus(ctx, dr, device.Name, provisioner.DeviceStatusFailed); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	l.V(1).Info("succeeded SetRequest", "response", prototext.Format(resp))
	oldDevice := device.DeepCopy()
	device.Status.Checksum = next.Checksum
	device.Status.LastApplied = newBuf

	if err := r.Status().Patch(ctx, device, client.MergeFrom(oldDevice)); err != nil {
		r.Error(ctx, err, "patch Device")
		return ctrl.Result{}, err
	}
	if err := r.updateRolloutStatus(ctx, dr, device.Name, provisioner.DeviceStatusCompleted); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DeviceReconciler) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := log.FromContext(ctx).WithCallDepth(1)
	if st := stacktrace.Get(err); st != "" {
		l = l.WithValues("stacktrace", st)
	}
	l.Error(err, msg, kvs...)
	// TODO Event emission
}

func (r *DeviceReconciler) updateRolloutStatus(ctx context.Context, dr provisioner.DeviceRollout, name string, status provisioner.DeviceStatus) error {
	oldDr := dr.DeepCopy()
	dr.Status.SetDeviceStatus(name, status)
	if err := r.Status().Patch(ctx, &dr, client.MergeFrom(oldDr)); err != nil {
		r.Error(ctx, err, "update DeviceRollout")
		return errors.WithStack(err)
	}
	return nil
}

func (r *DeviceReconciler) forceReplaceLastApplied(ctx context.Context, req ctrl.Request) error {
	device, err := r.getDevice(ctx, req.NamespacedName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	var dr provisioner.DeviceRollout
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: device.Namespace,
		Name:      device.Spec.RolloutRef,
	}, &dr); err != nil {
		return client.IgnoreNotFound(err)
	}

	var gr fluxcd.GitRepository
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: dr.Namespace,
		Name:      dr.Name,
	}, &gr); err != nil {
		return client.IgnoreNotFound(err)
	}

	dp, checksum, err := fetchArtifact(ctx, gr, device.Name, device.Spec.BaseRevision)
	if err != nil {
		return fmt.Errorf("fetch device config: %w", err)
	}
	defer os.RemoveAll(dp.RootDir)

	buf, err := dp.ReadDeviceConfigFile()
	if err != nil {
		return fmt.Errorf("read device config: %w", err)
	}

	old := device.DeepCopy()
	device.Status.LastApplied = buf
	device.Status.Checksum = checksum
	device.Status.BaseRevision = device.Spec.BaseRevision
	if err := r.Status().Patch(ctx, device, client.MergeFrom(old)); err != nil {
		return fmt.Errorf("patch DeviceRollout: %w", client.IgnoreNotFound(err))
	}
	return nil
}

func (r *DeviceReconciler) createSubscriberPodIfNotExist(ctx context.Context, nsName types.NamespacedName) error {
	d, err := r.getDevice(ctx, nsName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	subscriberPod := newSubscribePod(nsName, &d.Spec)
	var p core.Pod
	if err := r.Get(ctx, client.ObjectKeyFromObject(subscriberPod), &p); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get subscriber Pod: %w", err)
	}

	if err := ctrl.SetControllerReference(d, subscriberPod, r.Scheme); err != nil {
		return fmt.Errorf("create subscriber Pod: %w", err)
	}
	if err := r.Create(ctx, subscriberPod); err != nil {
		return fmt.Errorf("create subscriber subscriberPod: %w", err)
	}
	return nil
}

func (r *DeviceReconciler) findObjectForDeviceRollout(deviceRollout client.Object) []reconcile.Request {
	attachedDevices := deviceoperator.NewDeviceList()
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(device.RefField, deviceRollout.GetName()),
		Namespace:     deviceRollout.GetNamespace(),
	}

	ctx := context.TODO()
	if err := r.List(ctx, attachedDevices, listOps); err != nil {
		r.Error(ctx, err, "unable to list effected devices")
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedDevices.Items))
	for i, v := range attachedDevices.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      v.GetName(),
				Namespace: v.GetNamespace(),
			},
		}
	}
	return requests
}

func fetchArtifact(ctx context.Context, gr fluxcd.GitRepository, device, revision string) (*kuesta.DevicePath, string, error) {
	tmpDir, err := ioutil.TempDir("", gr.Name)
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}

	if revision == "" {
		_, err = artifact.FetchArtifact(ctx, gr, tmpDir)
	} else {
		_, err = artifact.FetchArtifactAt(ctx, gr, tmpDir, revision)
	}
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("fetch artifact: %w", err)
	}

	dp := &kuesta.DevicePath{RootDir: tmpDir, Device: device}
	checksum, err := dp.CheckSum()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", err
	}

	return dp, checksum, err
}

func decodeCueBytes(cctx *cue.Context, bytes []byte) (*model.Device, error) {
	val, err := kcue.NewValueFromBytes(cctx, bytes)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var o model.Device
	if err := val.Decode(&o); err != nil {
		return nil, errors.WithStack(err)
	}
	return &o, nil
}

func makeSetRequest(newBuf, curBuf []byte) (*gnmiproto.SetRequest, error) {
	cctx := cuecontext.New()

	newObj, err := decodeCueBytes(cctx, newBuf)
	if err != nil {
		return nil, fmt.Errorf("load new device config: %w", err)
	}
	curObj := &model.Device{}
	if curBuf != nil {
		curObj, err = decodeCueBytes(cctx, curBuf)
		if err != nil {
			return nil, fmt.Errorf("load current device config: %w", err)
		}
	}

	// TODO enhance performance
	n, err := ygot.Diff(curObj, newObj, &ygot.DiffPathOpt{
		MapToSinglePath: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get config diff: %w", err)
	}

	sr := &gnmiproto.SetRequest{
		Prefix: n.Prefix,
		Delete: n.Delete,
		Update: n.Update,
	}
	return sr, nil
}

func newSubscribePod(name types.NamespacedName, spec *device.DeviceSpec) *core.Pod {
	// TODO mount tls secret and add path to PEM files to env vars
	allowPrivilegeEscalation := false

	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("subscriber-%s", name.Name),
			Namespace: name.Namespace,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "kuesta-subscriber",
					Image:           subscriberImage + ":" + subscriberImageVersion,
					ImagePullPolicy: core.PullIfNotPresent,
					Command:         []string{"/bin/subscriber"},
					Env: []core.EnvVar{
						{Name: "KUESTA_DEVEL", Value: "true"},
						{Name: "KUESTA_VERBOSE", Value: "2"},
						{Name: "KUESTA_ADDR", Value: fmt.Sprintf("%s:%d", spec.Address, spec.Port)},
						{Name: "KUESTA_DEVICE", Value: name.Name},
						{Name: "KUESTA_AGGREGATOR_URL", Value: aggregatorUrl},
						{Name: "KUESTA_SKIP_VERIFY", Value: "true"},
					},
					SecurityContext: &core.SecurityContext{
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					},
				},
			},
		},
	}
}
