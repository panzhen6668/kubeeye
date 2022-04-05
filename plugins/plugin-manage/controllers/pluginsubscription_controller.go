/*
Copyright 2022.

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

package controllers

import (
	"context"
	"time"

	//"fmt"
	"github.com/kubesphere/kubeeye/pkg/expend"
	kubeeyev1alpha1 "github.com/kubesphere/kubeeye/plugins/plugin-manage/api/v1alpha1"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PluginSubscriptionReconciler reconciles a PluginSubscription object
type PluginSubscriptionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kubeeye.kubesphere.io,resources=pluginsubscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeeye.kubesphere.io,resources=pluginsubscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeeye.kubesphere.io,resources=pluginsubscriptions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PluginSubscription object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PluginSubscriptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)

	// TODO(user): your logic here
	logs.Info("PluginManageReconciler Reconcile !!!!!!!!!!!")
	pluginSub := &kubeeyev1alpha1.PluginSubscription{}

	err := r.Get(ctx, req.NamespacedName, pluginSub)
	if err != nil {
		if kubeErr.IsNotFound(err) {
			logs.Info("Cluster resource not found. Ignoring since object must be deleted ", "name", req.String())
			return ctrl.Result{}, nil
		}

		logs.Error(err, "Get pluginSub failed of ", req.String())
		return ctrl.Result{RequeueAfter: time.Second * 3}, err
	}

	finalizer := "kubeeye.kubesphere.io/plugin"
	if !pluginSub.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted, uninstall plugin
		logs.Info("Get pluginSub", "name", pluginSub.GetName())
		//fmt.Printf("3333GetName:%s !!!!!!!!!!!!!!!!! \n", pluginSub.GetName())
		if err := expend.UninstallPlugin(ctx, "", pluginSub.GetName()); err != nil {
			logs.Error(err, "Uninstall plugin failed ", "error:", err)
			return ctrl.Result{}, err
		}
		logs.Info("PluginManage uninstalled complete !!!!!!!!!!!!!!!!!!!!!!!!!!! !!!!!!!!!!!")
		pluginSub.ObjectMeta.Finalizers = removeString(pluginSub.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, pluginSub); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object.
	//append finalizer
	if !containsString(pluginSub.ObjectMeta.Finalizers, finalizer) {
		pluginSub.ObjectMeta.Finalizers = append(pluginSub.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, pluginSub); err != nil {
			logs.Error(err, "Update CR Status failed")
			return ctrl.Result{}, nil
		}
	}
	// if plugin is not installed, install it
	if !pluginSub.Status.Install {
		if err := expend.InstallPlugin(ctx, "", pluginSub.GetName()); err != nil {
			logs.Error(err, "Install KubeBench failed with error: %v")
			return ctrl.Result{}, err
		}
		logs.Info("PluginManage installed complete !!!!!!!!!!!!!!!!!!!!!!!!!!! !!!!!!!!!!!")
		// update install status
		if expend.IsPluginPodRunning() {
			pluginSub.Status.Install = true
			pluginSub.Status.Enabled = pluginSub.Spec.Enabled
			if err := r.Status().Update(ctx, pluginSub); err != nil {
				logs.Error(err, "Update CR Status failed")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PluginSubscriptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeeyev1alpha1.PluginSubscription{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
