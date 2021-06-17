/*
Copyright 2021 qinya0.

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
	"github.com/qinya0/k8s-operator-demo/service"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.qy.com,resources=appservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.qy.com,resources=appservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.qy.com,resources=appservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

func (r *AppServiceReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling AppService")

	// Fetch the AppService instance
	instance := &appv1.AppService{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp != nil {
		reqLogger.Info(" instance DeletionTimestamp")
		return reconcile.Result{}, err
	}

	// 如果不存在，则创建关联资源
	// 如果存在，判断是否需要更新
	//   如果需要更新，则直接更新
	//   如果不需要更新，则正常返回
	appIList := []service.AppInterface{
		&service.AppDeploy{},
	}

	for _, i := range appIList {
		serviceLog := reqLogger.WithValues("service", i.Name())

		if err := i.Init(&r.Client, instance, &ctx, request, &serviceLog); err != nil {
			serviceLog.Info("service init err:" + err.Error())
			return reconcile.Result{}, err
		}
		exist, err := i.Exist()
		if err != nil {
			serviceLog.Info("service exist err:" + err.Error())
			return reconcile.Result{}, err
		}
		if !exist {
			// not exist -> create
			if err = i.Create(); err != nil {
				serviceLog.Info("service create err:" + err.Error())
				return reconcile.Result{}, err
			}
			// create ok
			return reconcile.Result{}, nil
		}

		// exist -> check needUpdate
		needUpdate, err := i.NeedUpdate()
		if err != nil {
			serviceLog.Info("service needUpdate err:" + err.Error())
			return reconcile.Result{}, err
		}
		if !needUpdate {
			return reconcile.Result{}, nil
		}
		// update
		if err = i.Update(); err != nil {
			serviceLog.Info("service update err:" + err.Error())
			return reconcile.Result{}, err
		}
		serviceLog.Info("reconciling success")
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.AppService{}).
		Complete(r)
}
