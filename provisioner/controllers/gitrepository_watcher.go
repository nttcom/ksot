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
	"os"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/nttcom/kuesta/pkg/artifact"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/stacktrace"
	"github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GitRepositoryWatcher watches GitRepository objects for revision changes.
type GitRepositoryWatcher struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get
// +kubebuilder:rbac:groups=kuesta.hrk091.dev,resources=devicerollouts,verbs=get;list;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GitRepositoryWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconciliation")

	// get source object
	var repository sourcev1.GitRepository
	if err := r.Get(ctx, req.NamespacedName, &repository); err != nil {
		r.Error(ctx, err, "get GitRepository")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	l.Info(fmt.Sprintf("revision: %s", repository.Status.Artifact.Revision))

	tmpDir, err := os.MkdirTemp("", repository.Name)
	if err != nil {
		r.Error(ctx, err, "create temp dir")
		return ctrl.Result{}, err
	}
	defer os.RemoveAll(tmpDir)

	summary, err := artifact.FetchArtifact(ctx, repository, tmpDir)
	if err != nil {
		r.Error(ctx, err, "fetch artifact")
		return ctrl.Result{}, err
	}
	l.Info(summary)

	dps, err := kuesta.NewDevicePathList(tmpDir)
	if err != nil {
		r.Error(ctx, err, "list devices")
		return ctrl.Result{}, err
	}

	cmap := v1alpha1.DeviceConfigMap{}
	for _, dp := range dps {
		checksum, err := dp.CheckSum()
		if err != nil {
			r.Error(ctx, err, "get checksum")
			return ctrl.Result{}, err
		}
		cmap[dp.Device] = v1alpha1.DeviceConfig{
			Checksum:    checksum,
			GitRevision: repository.GetArtifact().Revision,
		}
	}

	dr := v1alpha1.DeviceRollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	if _, err = ctrl.CreateOrUpdate(ctx, r.Client, &dr, func() error {
		dr.Spec.DeviceConfigMap = cmap
		return nil
	}); err != nil {
		r.Error(ctx, err, "create or update DeviceRollout")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GitRepositoryWatcher) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	if err == nil {
		return
	}
	l := log.FromContext(ctx).WithCallDepth(1)
	if st := stacktrace.Get(err); st != "" {
		l = l.WithValues("stacktrace", st)
	}
	l.Error(err, msg, kvs...)
}

func (r *GitRepositoryWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sourcev1.GitRepository{}, builder.WithPredicates(GitRepositoryRevisionChangePredicate{})).
		Complete(r)
}
