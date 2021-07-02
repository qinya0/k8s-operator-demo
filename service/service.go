package service

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
	"github.com/qinya0/k8s-operator-demo/controllers"
	"github.com/qinya0/k8s-operator-demo/service/base"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AppService struct {
	base.BaseService
	Old *corev1.Service
}

func init() {
	controllers.RegistryInterface(&AppService{})
}

func (a *AppService) Name() string {
	return "Service"
}

func (a *AppService) Init(client *client.Client, appService *appv1.AppService, ctx *context.Context, request reconcile.Request, log *logr.Logger) error {
	_ = a.BaseService.Init(client, appService, ctx, request, log)

	// get svc
	svc := &corev1.Service{}
	if err := (*a.BaseService.Client).Get(context.TODO(), request.NamespacedName, svc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// not found
		a.Old = nil
	} else {
		a.Old = svc
		if a.Old.Annotations == nil {
			a.Old.Annotations = map[string]string{}
		}
	}
	return nil
}
func (a *AppService) Configure() bool {
	return a.App.Spec.Port.Port != 0
}
func (a *AppService) Exist() (bool, error) {

	return a.Old != nil, nil
}

func (a *AppService) Create() error {
	svc := a.newService()
	if err := (*a.Client).Create(context.TODO(), svc); err != nil {
		return err
	}
	(*a.Log).Info("create svc:" + a.App.Name)
	return nil
}

func (a *AppService) NeedUpdate() (bool, error) {
	if a.Old == nil {
		return false, errors.NewServiceUnavailable("no old svc,can't update")
	}
	svc := a.newService()

	oldSpec := appv1.AppServiceSpec{}
	if oldSpecStr, ok := a.Old.Annotations[base.AnnotationName]; ok && oldSpecStr != "" {
		if err := json.Unmarshal([]byte(oldSpecStr), &oldSpec); err != nil {
			return false, err
		}
	}

	if !reflect.DeepEqual(svc.Spec, oldSpec) {
		return true, nil
	}
	return false, nil
}
func (a *AppService) Update() error {
	svc := a.newService()
	a.Old.Spec = svc.Spec
	if err := (*a.Client).Update(context.TODO(), a.Old); err != nil {
		return err
	}
	(*a.Log).Info("update svc")
	return nil
}

func (a *AppService) Delete() (err error) {
	if a.Old == nil {
		return nil
	}
	err = (*a.Client).Delete(*a.Ctx, a.Old)
	if err != nil {
		return err
	}
	(*a.Log).Info("delete service")
	return
}

func (a *AppService) newService() *corev1.Service {
	spec := corev1.ServiceSpec{
		Type:  corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{a.App.Spec.Port},
		Selector: map[string]string{
			"app": a.App.Name,
		},
	}
	var annotations map[string]string
	if a.Old != nil {
		annotations = a.GenerateAnnotations(spec, a.Old.Annotations)
	} else {
		annotations = a.GenerateAnnotations(spec, nil)
	}

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
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
	if a.Old != nil && a.Old.Spec.ClusterIP != "" {
		svc.Spec.ClusterIP = a.Old.Spec.ClusterIP
	}
	return svc
}
