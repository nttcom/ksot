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

	deviceoperator "github.com/nttcom/kuesta/device-operator/api/v1alpha1"
	provisioner "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// OcDemoReconciler reconciles a OcDemo object.
type OcDemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	impl   *DeviceReconciler
}

//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=ocdemoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=ocdemoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=ocdemoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts,verbs=get;list;watch
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OcDemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	return r.impl.DoReconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OcDemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.SetupReconciler()
	return ctrl.NewControllerManagedBy(mgr).
		For(&deviceoperator.OcDemo{}).
		Owns(&corev1.Pod{}).
		Watches(
			&source.Kind{Type: &provisioner.DeviceRollout{}},
			handler.EnqueueRequestsFromMapFunc(r.impl.findObjectForDeviceRollout),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *OcDemoReconciler) SetupReconciler() {
	r.impl = &DeviceReconciler{r}
}

// DeviceReconciler reconciles a OcDemo object.
type DeviceReconciler struct {
	*OcDemoReconciler
}

func (r *DeviceReconciler) getDevice(ctx context.Context, nsName types.NamespacedName) (*deviceoperator.OcDemo, error) {
	device := deviceoperator.NewDevice()
	if err := r.Get(ctx, nsName, device); err != nil {
		return nil, errors.WithStack(err)
	}
	return device, nil
}
