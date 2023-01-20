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

	"github.com/nttcom/kuesta/pkg/stacktrace"
	kuestav1alpha1 "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeviceRolloutReconciler reconciles a DeviceRollout object.
type DeviceRolloutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DeviceRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconciliation")

	var dr kuestav1alpha1.DeviceRollout
	if err := r.Get(ctx, req.NamespacedName, &dr); err != nil {
		r.Error(ctx, err, "get DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	changed := dr.UpdateStatus()
	if changed {
		l.Info("changed", "phase", dr.Status.Phase, "status", dr.Status.Status)
		if err := r.Status().Update(ctx, &dr); err != nil {
			r.Error(ctx, err, "update DeviceRollout")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	} else {
		l.Info("not updated")
	}
	return ctrl.Result{}, nil
}

func (r *DeviceRolloutReconciler) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := log.FromContext(ctx).WithCallDepth(1)
	if st := stacktrace.Get(err); st != "" {
		l = l.WithValues("stacktrace", st)
	}
	l.Error(err, msg, kvs...)
	// TODO Event emission
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuestav1alpha1.DeviceRollout{}).
		Complete(r)
}
