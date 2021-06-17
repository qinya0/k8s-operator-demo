package service

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
	v1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AppDeploy struct {
	BaseService
	Old *appsv1.Deployment
}

func (a *AppDeploy) Name() string {
	return "Deployment"
}

func (a *AppDeploy) Init(client *client.Client, appService *appv1.AppService, ctx *context.Context, request reconcile.Request, log *logr.Logger) error {
	_ = a.BaseService.Init(client, appService, ctx, request, log)

	// get deploy
	deploy := &appsv1.Deployment{}
	if err := (*a.BaseService.Client).Get(context.TODO(), request.NamespacedName, deploy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// not found
		a.Old = nil
	} else {
		a.Old = deploy
		if a.Old.Annotations == nil {
			a.Old.Annotations = map[string]string{}
		}
	}
	return nil
}

func (a *AppDeploy) Exist() (bool, error) {
	return a.Old != nil, nil
}

func (a *AppDeploy) Create() error {
	deploy := a.newDeploy(a.App, nil)
	if err := (*a.Client).Create(context.TODO(), deploy); err != nil {
		return err
	}
	(*a.Log).Info("create deploy:" + a.App.Name)
	return nil
}
func (a *AppDeploy) NeedUpdate() (bool, error) {
	if a.Old == nil {
		return false, errors.NewServiceUnavailable("no old deployment,can't update")
	}
	deploy := a.newDeploy(a.App, a.Old.Annotations)

	oldSpec := appv1.AppServiceSpec{}
	if oldSpecStr, ok := a.Old.Annotations[AnnotationName]; ok && oldSpecStr != "" {
		if err := json.Unmarshal([]byte(oldSpecStr), &oldSpec); err != nil {
			return false, err
		}
	}

	if !reflect.DeepEqual(deploy.Spec, oldSpec) {
		return true, nil
	}
	return false, nil
}
func (a *AppDeploy) Update() error {
	deploy := a.newDeploy(a.App, a.Old.Annotations)
	a.Old.Spec = deploy.Spec
	if err := (*a.Client).Update(context.TODO(), a.Old); err != nil {
		return err
	}
	(*a.Log).Info("update deploy")
	return nil
}

func (a *AppDeploy) newDeploy(app *appv1.AppService, annotations map[string]string) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}

	spec := appsv1.DeploymentSpec{
		Replicas: &app.Spec.Size,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
		Selector: selector,
	}

	annotations = a.generateAnnotations(spec, annotations)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    "AppService",
				}),
			},
		},
		Spec: spec,
	}
}

func newContainers(app *appv1.AppService) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts, cport)
	}
	return []corev1.Container{
		{
			Name:            app.Name,
			Image:           app.Spec.Image,
			Resources:       app.Spec.Resources,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             app.Spec.Envs,
		},
	}
}
