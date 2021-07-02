package service

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
	"github.com/qinya0/k8s-operator-demo/controllers"
	"github.com/qinya0/k8s-operator-demo/service/base"
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
	base.BaseService
	Old *appsv1.Deployment
}

func init() {
	controllers.RegistryInterface(&AppDeploy{})
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

func (a *AppDeploy) Configure() bool {
	return true
}
func (a *AppDeploy) Exist() (bool, error) {
	return a.Old != nil, nil
}

func (a *AppDeploy) Create() error {
	deploy := a.newDeploy()
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
	deploy := a.newDeploy()

	oldSpec := appv1.AppServiceSpec{}
	if oldSpecStr, ok := a.Old.Annotations[base.AnnotationName]; ok && oldSpecStr != "" {
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
	deploy := a.newDeploy()
	a.Old.Spec = deploy.Spec
	if err := (*a.Client).Update(context.TODO(), a.Old); err != nil {
		return err
	}
	(*a.Log).Info("update deploy")
	return nil
}

func (a *AppDeploy) Delete() (err error) {
	if a.Old == nil {
		return nil
	}
	err = (*a.Client).Delete(*a.Ctx, a.Old)
	if err != nil {
		return err
	}
	(*a.Log).Info("delete deploy")
	return
}

func (a *AppDeploy) newDeploy() *appsv1.Deployment {
	labels := map[string]string{"app": a.App.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}

	spec := appsv1.DeploymentSpec{
		Replicas: &a.App.Spec.Size,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: a.newContainers(),
			},
		},
		Selector: selector,
	}

	var annotations map[string]string
	if a.Old != nil {
		annotations = a.GenerateAnnotations(spec, a.Old.Annotations)
	} else {
		annotations = a.GenerateAnnotations(spec, nil)
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.App.Name,
			Namespace:   a.App.Namespace,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(a.App, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    "AppService",
				}),
			},
		},
		Spec: spec,
	}
}

func (a *AppDeploy) newContainers() []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	if a.App.Spec.Port.Port != 0 {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = a.App.Spec.Port.TargetPort.IntVal
		containerPorts = append(containerPorts, cport)
	}
	return []corev1.Container{
		{
			Name:            a.App.Name,
			Image:           a.App.Spec.Image,
			Resources:       a.App.Spec.Resources,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             a.App.Spec.Envs,
		},
	}
}
